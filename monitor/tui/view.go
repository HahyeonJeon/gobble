package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/HahyeonJeon/gobble/monitor"
)

const headerHeight = 9

func (m *model) bodyHeight() int { return max(1,m.height-headerHeight-1) }

func (m *model) View() tea.View {
	var lines []string
	if m.width<44 || m.height<16 {
		lines=[]string{"GOBBLE · "+oneLine(m.data.Snapshot.Run.Status),fmt.Sprintf("Tasks: %d/%d succeeded",m.data.Total.Succeeded,m.data.Total.Total),fmt.Sprintf("Attention: %d",m.data.Total.Attention()),"Resize to at least 44 × 16", "q quit monitor · execution continues"}
	} else {
		lines=m.header()
		var body []string
		switch m.screen {
		case searchScreen: body=m.searchView()
		case tasksScreen,attentionScreen: body=m.tasksView()
		case detailScreen: body=m.detailView()
		case helpScreen: body=m.helpView()
		default: body=m.dashboardView()
		}
		lines=append(lines,frame(body,m.width,m.bodyHeight())...)
		lines=append(lines,m.footer())
	}
	v:=tea.NewView(strings.Join(frame(lines,m.width,m.height),"\n"))
	v.AltScreen=true
	if !m.style.monochrome {
		v.BackgroundColor=lipgloss.Color("#10171B")
		v.ForegroundColor=lipgloss.Color("#DBE8EC")
	}
	return v
}

func frame(lines []string,width,height int) []string {
	out:=make([]string,height)
	for i:=range out {
		if i<len(lines) {out[i]=fit(lines[i],width)} else {out[i]=strings.Repeat(" ",width)}
	}
	return out
}

func elapsed(start,end string,now time.Time) string {
	t,err:=time.Parse(time.RFC3339Nano,start)
	if err!=nil {return "—"}
	if end!="" {if e,err:=time.Parse(time.RFC3339Nano,end);err==nil {now=e}}
	d:=max(time.Duration(0),now.Sub(t)).Truncate(time.Second)
	return d.String()
}

func (m *model) observationTime() time.Time {
	if m.err!=nil || !m.data.Snapshot.Run.Occupancy.Live {return m.data.Snapshot.ReadAt}
	return m.now
}

func (m *model) header() []string {
	c:=m.data.Total;s:=m.data.Snapshot
	complete:=0
	for _,sample:=range m.data.Samples {if sample.Counts.Successful(){complete++}}
	scope:=fmt.Sprintf("All samples · %d/%d sample task sets complete · shared/unassigned %d/%d",complete,len(m.data.Samples),m.data.Shared.Succeeded,m.data.Shared.Total)
	if m.sample!="" {sc:=m.data.Count(m.data.SampleTasks(m.sample));scope=fmt.Sprintf("Sample %s · %d/%d owned tasks succeeded · shared work is context",oneLine(m.sample),sc.Succeeded,sc.Total)}
	fresh:=fmt.Sprintf("Last read %s · read-only · q leaves pipeline running",s.ReadAt.Format("15:04:05"))
	if m.err!=nil {fresh=m.style.bad.Render("STALE · "+oneLine(m.err.Error())+" · last read "+s.ReadAt.Format("15:04:05"))
	} else if s.Run.Status=="running" && !s.Run.Occupancy.Live {fresh=m.style.warn.Render("OWNER NOT LIVE · recorded run status may be outdated · inspect before recovery")
	} else if s.Run.Unknown {fresh=m.style.warn.Render("BACKEND UNCONFIRMED · inspect affected work before recovery")}
	name:=s.Pipeline;if name=="" {name=s.Run.ID}
	primary:=fmt.Sprintf("Succeeded %d/%d (%.0f%% known)  Running %d  Failed %d",c.Succeeded,c.Total,c.Percent(),c.Running,c.Failed)
	secondary:=fmt.Sprintf("Pending %d  Blocked %d  Skipped %d  Incomplete %d",c.Pending,c.Blocked,c.Skipped,c.Incomplete)
	tertiary:=fmt.Sprintf("Unknown %d  Unfinalized %d  Reused %d  Expanding %d",c.Unknown,c.Unfinalized,c.Reused,c.Unexpanded)
	if m.width < 70 {
		primary=fmt.Sprintf("OK %d/%d (%.0f%%)  Run %d  Fail %d",c.Succeeded,c.Total,c.Percent(),c.Running,c.Failed)
		secondary=fmt.Sprintf("Wait %d  Block %d  Skip %d  Incomp %d",c.Pending,c.Blocked,c.Skipped,c.Incomplete)
		tertiary=fmt.Sprintf("Unknown %d  Unfinal %d  Reuse %d  Expand %d",c.Unknown,c.Unfinalized,c.Reused,c.Unexpanded)
	}
	return []string{
		m.style.active.Bold(true).Render("G O B B L E")+m.style.dim.Render(" / pipeline monitor"),
		oneLine(name)+" · "+m.style.state(s.Run.Status).Render(oneLine(s.Run.Status))+" · "+elapsed(s.Run.Started,s.Run.Ended,m.observationTime()),
		m.style.dim.Render(oneLine(m.workspace)),
		primary,
		m.style.dim.Render(secondary),
		m.style.dim.Render(tertiary),
		m.style.active.Render(scope),fresh,m.style.dim.Render(strings.Repeat("─",m.width)),
	}
}

func (m *model) dashboardView() []string {
	width:=m.graphWidth();h:=m.bodyHeight()-1
	graph:=m.graph(width,max(1,h))
	if m.width>=100 {
		side:=frame(m.sidebar(),30,h)
		for i:=range graph {graph[i]+=m.style.dim.Render(" │")+side[i]}
	}
	return append([]string{m.style.dim.Render("PIPELINE GRAPH · ↑↓ select · Enter tasks · PgUp/PgDn pan · ! attention")},graph...)
}

func (m *model) sidebar() []string {
	lines:=[]string{m.style.bad.Render(fmt.Sprintf("ATTENTION · %d tasks",len(m.data.Attention)))}
	for i,index:=range m.data.Attention {
		if i==3 {lines=append(lines,fmt.Sprintf("+ %d more · press !",len(m.data.Attention)-3));break}
		t:=m.data.Snapshot.Tasks[index]
		lines=append(lines,m.style.state(t.Status).Render(oneLine(t.Status)),oneLine(t.Identity),m.style.dim.Render(oneLine(t.Reason)),"")
	}
	if len(m.data.Attention)==0 {lines=append(lines,m.style.good.Render("No task failures recorded"),"")}
	i:=m.stageIndex()
	if i>=0 {
		s:=m.data.Stages[i];counts:=m.data.Count(m.data.StageTasks(s,m.sample));up,down:=m.data.Neighbors(s.ID)
		lines=append(lines,m.style.active.Render("SELECTED STAGE"),oneLine(s.Name),fmt.Sprintf("%d/%d succeeded",counts.Succeeded,counts.Total))
		lines=append(lines,wrapPlain(nodeStatus(counts),30)...)
		lines=append(lines,"",m.style.dim.Render("UPSTREAM"))
		if len(up)==0 {lines=append(lines,"Pipeline inputs")}
		for _,name:=range up {lines=append(lines,oneLine(name))}
		lines=append(lines,m.style.dim.Render("DOWNSTREAM"))
		for _,name:=range down {lines=append(lines,oneLine(name))}
	}
	return lines
}

func (m *model) searchView() []string {
	lines:=[]string{m.style.active.Render("FIND SAMPLE"),"> "+oneLine(m.query)+"▏",""}
	matches:=m.data.SearchSamples(m.query)
	if len(matches)==0 {
		message:="No matching samples"
		if len(m.data.Samples)==0 {message="This run has no sample labels. Task navigation remains available."}
		return append(lines,wrapPlain(message,m.width)...)
	}
	start:=max(0,m.searchIndex-max(1,m.bodyHeight()-5)+1)
	for i:=start;i<len(matches)&&len(lines)<m.bodyHeight()-1;i++ {
		s:=matches[i];row:=fmt.Sprintf("%s  %d/%d succeeded · %s",oneLine(s.ID),s.Counts.Succeeded,s.Counts.Total,nodeStatus(s.Counts))
		if i==m.searchIndex {row=m.style.selected.Render(fit("▸ "+row,m.width))}else{row="  "+row}
		lines=append(lines,row)
	}
	return append(lines,m.style.dim.Render(fmt.Sprintf("%d matches · Enter selects an exact identity · Esc returns",len(matches))))
}

func (m *model) tasksView() []string {
	label:="TASKS"
	if m.screen==attentionScreen {label="ATTENTION · all samples"}
	items:=m.listTasks()
	lines:=[]string{m.style.active.Render(fmt.Sprintf("%s · %d instances",label,len(items))),m.style.dim.Render(fit("TASK",m.width-27)+fit("STATUS",18)+"ELAPSED")}
	if len(items)==0 {return append(lines,"No tasks in this selection")}
	start:=max(0,m.listIndex-max(1,m.bodyHeight()-4)+1)
	for i:=start;i<len(items)&&len(lines)<m.bodyHeight()-1;i++ {
		t:=m.data.Snapshot.Tasks[items[i]]
		status:=t.Status;if t.Template {status="template"}
		row:=fit(oneLine(t.Identity),m.width-27)+m.style.state(t.Status).Render(fit(status,18))+elapsed(t.Started,t.Ended,m.observationTime())
		if i==m.listIndex {row=m.style.selected.Render(fit(row,m.width))}
		lines=append(lines,row)
	}
	selected:=m.data.Snapshot.Tasks[items[min(m.listIndex,len(items)-1)]]
	return append(lines,m.style.dim.Render(oneLine(selected.Reason)))
}

func (m *model) currentLog() (string,int64) {
	for _,log:=range m.data.Snapshot.Logs {
		if log.Identity==m.task {
			if m.logStream=="stdout" {return log.StdoutTail,log.StdoutSize}
			return log.StderrTail,log.StderrSize
		}
	}
	return "",0
}

func (m *model) logLines() []string {
	text,_:=m.currentLog()
	if text=="" {text="No output available in this tail."}
	return wrapPlain(strings.ReplaceAll(clean(text),"\t","    "),m.width)
}

func (m *model) logTailOffset() int {return max(0,len(m.logLines())-max(1,m.bodyHeight()-7))}

func (m *model) detailView() []string {
	t,ok:=m.data.Task(m.task);if !ok{return []string{"Task is no longer in this snapshot."}}
	_,size:=m.currentLog()
	follow:="paused";if m.follow {follow="following"}
	command:=strings.Join(t.Command," ");if t.Script!="" {command=t.Script}
	lines:=[]string{
		m.style.active.Render(oneLine(t.Identity)),
		fmt.Sprintf("%s · attempt %d · %s · CPU request %.1f · RAM request %s",oneLine(t.Status),t.Attempt,oneLine(t.Executor),t.Resources.CPU,oneLine(t.Resources.Memory)),
		m.style.dim.Render("Image: "+oneLine(t.Image)),
		"Command: "+oneLine(command),
		m.style.state(t.Status).Render("Reason: "+oneLine(t.Reason)),
		m.style.active.Render(fmt.Sprintf("1 stdout / 2 stderr · %s · %s · tail ≤4 KiB of %d bytes",m.logStream,follow,size)),
		m.style.dim.Render(strings.Repeat("─",m.width)),
	}
	logs:=m.logLines();offset:=min(m.logOffset,m.logTailOffset());if m.follow {offset=m.logTailOffset()}
	return append(lines,logs[offset:]...)
}

func wrapPlain(text string,width int) []string {
	width=max(1,width);lines:=[]string{}
	for _,line:=range strings.Split(clean(text),"\n") {
		var out strings.Builder;used:=0
		for _,r:=range line {
			w:=lipgloss.Width(string(r))
			if used+w>width {lines=append(lines,out.String());out.Reset();used=0}
			out.WriteRune(r);used+=w
		}
		lines=append(lines,out.String())
	}
	return lines
}

func (m *model) helpView() []string {
	return []string{"KEYBOARD","","↑ ↓ / j k     Select stage, sample, or task","Enter         Open stage tasks, task details, or sample","/ or s        Find sample; Enter selects its exact ID","t             Tasks in current sample scope","!             Global attention list","PgUp / PgDn   Pan graph or scroll lists/logs","1 / 2         Switch stdout / stderr in task details","f / End       Toggle follow / follow newest log tail","r             Refresh now","Esc           Back; on dashboard, clear sample scope","q / Ctrl+C    Close monitor; pipeline keeps running","","Counts describe known tasks, not remaining compute time.","Shared/cohort tasks are excluded from sample completion.","Log history is bounded to the last 4 KiB per stream.","Unknown and unfinalized states require inspection.","NO_COLOR disables colors. Terminal resizing preserves focus."}
}

func (m *model) footer() string {
	return m.style.dim.Render("/ sample  ! attention  Enter open  Esc back  ? help  q quit monitor")
}
