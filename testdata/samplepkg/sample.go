package samplepkg

import "context"

type UserCfg struct {
	userNum int
}

func Handle(ctx context.Context, userNum int) int {
	resultVal := userNum + 1

	for idx, usr := range []string{"a", "b"} {
		_ = idx
		_ = usr
	}

	_ = ctx

	return resultVal
}
