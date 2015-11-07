package ssh

import (
	"fmt"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SuiteCommon struct{}

var _ = Suite(&SuiteCommon{})

func (s *SuiteRemote) TestKeyboardInteractiveName(c *C) {
	a := &KeyboardInteractive{
		user:      "test",
		challenge: nil,
	}
	c.Assert(a.Name(), Equals, KeyboardInteractiveName)
}

func (s *SuiteRemote) TestKeyboardInteractiveString(c *C) {
	a := &KeyboardInteractive{
		user:      "test",
		challenge: nil,
	}
	c.Assert(a.String(), Equals, fmt.Sprintf("user: test, name: %s", KeyboardInteractiveName))
}

func (s *SuiteRemote) TestPasswordName(c *C) {
	a := &Password{
		user: "test",
		pass: "",
	}
	c.Assert(a.Name(), Equals, PasswordName)
}

func (s *SuiteRemote) TestPasswordString(c *C) {
	a := &Password{
		user: "test",
		pass: "",
	}
	c.Assert(a.String(), Equals, fmt.Sprintf("user: test, name: %s", PasswordName))
}

func (s *SuiteRemote) TestPasswordCallbackName(c *C) {
	a := &PasswordCallback{
		user:     "test",
		callback: nil,
	}
	c.Assert(a.Name(), Equals, PasswordCallbackName)
}

func (s *SuiteRemote) TestPasswordCallbackString(c *C) {
	a := &PasswordCallback{
		user:     "test",
		callback: nil,
	}
	c.Assert(a.String(), Equals, fmt.Sprintf("user: test, name: %s", PasswordCallbackName))
}

func (s *SuiteRemote) TestPublicKeysName(c *C) {
	a := &PublicKeys{
		user:   "test",
		signer: nil,
	}
	c.Assert(a.Name(), Equals, PublicKeysName)
}

func (s *SuiteRemote) TestPublicKeysString(c *C) {
	a := &PublicKeys{
		user:   "test",
		signer: nil,
	}
	c.Assert(a.String(), Equals, fmt.Sprintf("user: test, name: %s", PublicKeysName))
}

func (s *SuiteRemote) TestPublicKeysCallbackName(c *C) {
	a := &PublicKeysCallback{
		user:     "test",
		callback: nil,
	}
	c.Assert(a.Name(), Equals, PublicKeysCallbackName)
}

func (s *SuiteRemote) TestPublicKeysCallbackString(c *C) {
	a := &PublicKeysCallback{
		user:     "test",
		callback: nil,
	}
	c.Assert(a.String(), Equals, fmt.Sprintf("user: test, name: %s", PublicKeysCallbackName))
}
