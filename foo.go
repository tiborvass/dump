package dump

import (
	"fmt"

	"github.com/docker/docker/api/errors"
)

func Foo() {
	fmt.Println(errors.NewBadRequestError(fmt.Errorf("foo")))
}
