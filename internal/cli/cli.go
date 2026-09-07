package cli

import "fmt"

func Greet(name string) error {
	fmt.Printf("Hello, %s!\n", name)

	return nil
}
