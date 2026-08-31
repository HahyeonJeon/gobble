package gatk4haplotypecaller_test

import "github.com/HahyeonJeon/gobble/assets/modules"

func moduleOptions(extra []string) modules.Options {
	return modules.Options{ExtraArgs: append([]string(nil), extra...)}
}
