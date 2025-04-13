package utils

func Assert(predicate bool, message string) {
	if predicate {
		panic(message)
	}
}
