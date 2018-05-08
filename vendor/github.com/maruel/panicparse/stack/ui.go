
package stack

import (
	"fmt"
	"strings"
)

type Palette struct {
	EOLReset string

	RoutineFirst string 
	Routine      string 
	CreatedBy    string

	Package                string
	SourceFile             string
	FunctionStdLib         string
	FunctionStdLibExported string
	FunctionMain           string
	FunctionOther          string
	FunctionOtherExported  string
	Arguments              string
}

func CalcLengths(buckets Buckets, fullPath bool) (int, int) {
	srcLen := 0
	pkgLen := 0
	for _, bucket := range buckets {
		for _, line := range bucket.Signature.Stack.Calls {
			l := 0
			if fullPath {
				l = len(line.FullSourceLine())
			} else {
				l = len(line.SourceLine())
			}
			if l > srcLen {
				srcLen = l
			}
			l = len(line.Func.PkgName())
			if l > pkgLen {
				pkgLen = l
			}
		}
	}
	return srcLen, pkgLen
}

func (p *Palette) functionColor(line *Call) string {
	if line.IsStdlib() {
		if line.Func.IsExported() {
			return p.FunctionStdLibExported
		}
		return p.FunctionStdLib
	} else if line.IsPkgMain() {
		return p.FunctionMain
	} else if line.Func.IsExported() {
		return p.FunctionOtherExported
	}
	return p.FunctionOther
}

func (p *Palette) routineColor(bucket *Bucket, multipleBuckets bool) string {
	if bucket.First() && multipleBuckets {
		return p.RoutineFirst
	}
	return p.Routine
}

func (p *Palette) BucketHeader(bucket *Bucket, fullPath, multipleBuckets bool) string {
	extra := ""
	if bucket.SleepMax != 0 {
		if bucket.SleepMin != bucket.SleepMax {
			extra += fmt.Sprintf(" [%d~%d minutes]", bucket.SleepMin, bucket.SleepMax)
		} else {
			extra += fmt.Sprintf(" [%d minutes]", bucket.SleepMax)
		}
	}
	if bucket.Locked {
		extra += " [locked]"
	}
	created := bucket.CreatedBy.Func.PkgDotName()
	if created != "" {
		created += " @ "
		if fullPath {
			created += bucket.CreatedBy.FullSourceLine()
		} else {
			created += bucket.CreatedBy.SourceLine()
		}
		extra += p.CreatedBy + " [Created by " + created + "]"
	}
	return fmt.Sprintf(
		"%s%d: %s%s%s\n",
		p.routineColor(bucket, multipleBuckets), len(bucket.Routines),
		bucket.State, extra,
		p.EOLReset)
}

func (p *Palette) callLine(line *Call, srcLen, pkgLen int, fullPath bool) string {
	src := ""
	if fullPath {
		src = line.FullSourceLine()
	} else {
		src = line.SourceLine()
	}
	return fmt.Sprintf(
		"    %s%-*s %s%-*s %s%s%s(%s)%s",
		p.Package, pkgLen, line.Func.PkgName(),
		p.SourceFile, srcLen, src,
		p.functionColor(line), line.Func.Name(),
		p.Arguments, line.Args,
		p.EOLReset)
}

func (p *Palette) StackLines(signature *Signature, srcLen, pkgLen int, fullPath bool) string {
	out := make([]string, len(signature.Stack.Calls))
	for i := range signature.Stack.Calls {
		out[i] = p.callLine(&signature.Stack.Calls[i], srcLen, pkgLen, fullPath)
	}
	if signature.Stack.Elided {
		out = append(out, "    (...)")
	}
	return strings.Join(out, "\n") + "\n"
}
