package ssh

import (
	"fmt"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SuiteCommon struct{}

var _ = Suite(&SuiteCommon{})

func (s *SuiteRemote) TestPublicKeysCallbackName(c *C) {
	a := &PublicKeysCallback{
		user:    "test",
		setAuth: nil,
	}
	c.Assert(a.Name(), Equals, PublicKeysCallbackName)
}

func (s *SuiteRemote) TestPublicKeysCallbackString(c *C) {
	a := &PublicKeysCallback{
		user:    "test",
		setAuth: nil,
	}
	c.Assert(a.String(), Equals, fmt.Sprintf("user: test, name: %s", PublicKeysCallbackName))
}
