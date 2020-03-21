package main

type Args struct {
	Output *string
	Overwrite *string
}

func (args *Args) CheckOverwrite() bool {
	overwrite := *args.Overwrite
	if overwrite == `fail` { return true }
	if overwrite == `uniq` { return true }
	if overwrite == `overwrite` { return true }
	return false
}
