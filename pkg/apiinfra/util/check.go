package util

// Check ensures an error is handled by panicking if the error is not nil.
func Check(err error) {
	if err != nil {
		panic(err)
	}
}

// CheckWrite checks for an error and ensures it is handled using the Check function.
func CheckWrite(_ int, err error) {
	Check(err)
}
