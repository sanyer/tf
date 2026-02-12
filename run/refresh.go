package run

func Refresh(args []string) error {
	return terraformWithProgress("refresh", args)
}
