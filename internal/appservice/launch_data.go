package appservice

import (
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"
)

const maxLaunchInput int64 = 64 << 30
const maxLaunchDecoded int64 = 256 << 30
const maxReadLine = 16 << 20

type copyProgress struct {
	ctx    context.Context
	dst    io.Writer
	total  int64
	report func(int64)
}

func (p *copyProgress) Write(b []byte) (int, error) {
	if err := p.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := p.dst.Write(b)
	p.total += int64(n)
	p.report(p.total)
	return n, err
}

// copyFASTQ validates a bounded four-line dialect while retaining EXACT input
// bytes (compressed when supplied). Gzip checksum/truncation must reach EOF.
func copyFASTQ(ctx context.Context, src io.Reader, dst *os.File, compressed bool, progress func(int64)) (string, int64, int64, error) {
	sum := sha256.New()
	p := &copyProgress{ctx: ctx, dst: io.MultiWriter(dst, sum), report: progress}
	counted := io.TeeReader(io.LimitReader(src, maxLaunchInput+1), p)
	var decoded io.Reader = counted
	if compressed {
		gz, err := gzip.NewReader(counted)
		if err != nil {
			return "", 0, 0, problem("invalid_request", "The gzip read file is invalid or incomplete.")
		}
		defer gz.Close()
		decoded = gz
	}
	limited := &io.LimitedReader{R: decoded, N: maxLaunchDecoded + 1}
	scanner := bufio.NewScanner(limited)
	scanner.Buffer(make([]byte, 64<<10), maxReadLine)
	line, reads, sequenceLength := 0, int64(0), 0
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return "", p.total, reads, err
		}
		b := scanner.Bytes()
		if len(b) > 0 && b[len(b)-1] == '\r' {
			b = b[:len(b)-1]
		}
		valid := true
		switch line % 4 {
		case 0:
			valid = len(b) > 1 && b[0] == '@'
		case 1:
			sequenceLength = len(b)
			valid = sequenceLength > 0
			for _, c := range b {
				if !strings.ContainsRune("ACGTUNRYKMSWBDHVacgtunrykmswbdhv", rune(c)) {
					valid = false
					break
				}
			}
		case 2:
			valid = len(b) > 0 && b[0] == '+'
		case 3:
			valid = len(b) == sequenceLength
			for _, c := range b {
				if c < 33 || c > 126 {
					valid = false
					break
				}
			}
			reads++
		}
		if !valid {
			return "", p.total, reads, problem("invalid_request", "Use complete four-line FASTQ records with matching sequence and quality lengths.")
		}
		line++
	}
	if scanner.Err() != nil || limited.N <= 0 || p.total > maxLaunchInput || line%4 != 0 || reads == 0 {
		return "", p.total, reads, problem("invalid_request", "The read file is incomplete, empty, or exceeds the supported FASTQ bounds.")
	}
	if err := dst.Sync(); err != nil {
		return "", p.total, reads, err
	}
	return fmt.Sprintf("sha256:%x", sum.Sum(nil)), p.total, reads, nil
}

func hashLaunchInput(ctx context.Context, src io.Reader) (string, error) {
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(launchInputReader{ctx, src}, maxLaunchInput+1))
	if err != nil {
		return "", err
	}
	if n > maxLaunchInput {
		return "", io.ErrShortBuffer
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}

type launchInputReader struct {
	ctx context.Context
	io.Reader
}

func (r launchInputReader) Read(b []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.Reader.Read(b)
}
