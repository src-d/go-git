package advrefs_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"gopkg.in/src-d/go-git.v3/clients/common"
	"gopkg.in/src-d/go-git.v3/core"
	"gopkg.in/src-d/go-git.v3/formats/packp/advrefs"
	"gopkg.in/src-d/go-git.v3/formats/packp/pktline"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SuiteAdvRefs struct{}

var _ = Suite(&SuiteAdvRefs{})

func (s *SuiteAdvRefs) TestParseEncode(c *C) {
	for _, test := range [...]struct {
		input    []string
		expected []string
	}{
		{
			input: []string{
				"0000000000000000000000000000000000000000 capabilities^{}\x00",
				"",
			},
			expected: []string{
				"0000000000000000000000000000000000000000 capabilities^{}\x00\n",
				"",
			},
		}, {
			input: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00",
				"",
			},
			expected: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00\n",
				"",
			},
		}, {
			input: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00symref=HEAD:/refs/heads/master ofs-delta multi_ack",
				"a6930aaee06755d1bdcfd943fbf614e4d92bb0c7 refs/heads/master",
				"5dc01c595e6c6ec9ccda4f6f69c131c0dd945f8c refs/tags/v2.6.11-tree",
				"c39ae07f393806ccf406ef966e9a15afc43cc36a refs/tags/v2.6.11-tree^{}",
				"",
			},
			expected: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00multi_ack ofs-delta symref=HEAD:/refs/heads/master\n",
				"a6930aaee06755d1bdcfd943fbf614e4d92bb0c7 refs/heads/master\n",
				"5dc01c595e6c6ec9ccda4f6f69c131c0dd945f8c refs/tags/v2.6.11-tree\n",
				"c39ae07f393806ccf406ef966e9a15afc43cc36a refs/tags/v2.6.11-tree^{}\n",
				"",
			},
		}, {
			input: []string{
				"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00multi_ack symref=HEAD:/refs/heads/master ofs-delta\n",
				"a6930aaee06755d1bdcfd943fbf614e4d92bb0c7 refs/heads/master\n",
				"5dc01c595e6c6ec9ccda4f6f69c131c0dd945f8c refs/tags/v2.6.11-tree\n",
				"c39ae07f393806ccf406ef966e9a15afc43cc36a refs/tags/v2.6.11-tree^{}\n",
				"shallow 1111111111111111111111111111111111111111\n",
				"shallow 2222222222222222222222222222222222222222\n",
				"",
			},
			expected: []string{"6ecf0ef2c2dffb796033e5a02219af86ec6584e5 HEAD\x00multi_ack ofs-delta symref=HEAD:/refs/heads/master\n",
				"a6930aaee06755d1bdcfd943fbf614e4d92bb0c7 refs/heads/master\n",
				"5dc01c595e6c6ec9ccda4f6f69c131c0dd945f8c refs/tags/v2.6.11-tree\n",
				"c39ae07f393806ccf406ef966e9a15afc43cc36a refs/tags/v2.6.11-tree^{}\n",
				"shallow 1111111111111111111111111111111111111111\n",
				"shallow 2222222222222222222222222222222222222222\n",
				"",
			},
		},
	} {
		var err error

		var input bytes.Buffer
		var comment CommentInterface
		{
			r, err := pktline.NewFromStrings(test.input...)
			c.Assert(err, IsNil, Commentf("input = %v\n", test.input))
			tee := io.TeeReader(r, &input)
			inputCopy, err := ioutil.ReadAll(tee)
			c.Assert(err, IsNil, Commentf("input = %v\n", test.input))
			comment = Commentf("input = %s\n", string(inputCopy))
		}

		var expected []byte
		{
			r, err := pktline.NewFromStrings(test.expected...)
			c.Assert(err, IsNil, comment)
			expected, err = ioutil.ReadAll(r)
			c.Assert(err, IsNil, comment)
		}

		var ar *advrefs.Contents
		{
			ar, err = advrefs.Parse(&input)
			c.Assert(err, IsNil, comment)
		}

		var obtained []byte
		{
			r, err := ar.Encode()
			c.Assert(err, IsNil)
			obtained, err = ioutil.ReadAll(r)
			c.Assert(err, IsNil, comment)
		}

		c.Assert(obtained, DeepEquals, expected, comment)
	}
}

func ExampleParse() {
	// Here is a raw advertised-ref message.
	input := strings.NewReader(
		"0065a6930aaee06755d1bdcfd943fbf614e4d92bb0c7 HEAD\x00multi_ack ofs-delta symref=HEAD:/refs/heads/master\n" +
			"003fa6930aaee06755d1bdcfd943fbf614e4d92bb0c7 refs/heads/master\n" +
			"00441111111111111111111111111111111111111111 refs/tags/v2.6.11-tree\n" +
			"00475555555555555555555555555555555555555555 refs/tags/v2.6.11-tree^{}\n" +
			"0035shallow 5dc01c595e6c6ec9ccda4f6f69c131c0dd945f8c\n" +
			"0000")

	// Once parsed...
	ar, _ := advrefs.Parse(input)

	//... you can access all the data in the original message
	fmt.Println("head =", ar.Head)
	fmt.Println("caps =", ar.Caps.String())
	fmt.Println("...")
	fmt.Println("shallows =", ar.Shallows)
	// Output: head = a6930aaee06755d1bdcfd943fbf614e4d92bb0c7
	// caps = multi_ack ofs-delta symref=HEAD:/refs/heads/master
	// ...
	// shallows = [5dc01c595e6c6ec9ccda4f6f69c131c0dd945f8c]
}

func ExampleContents_Encode() {
	head := core.NewHash("1111111111111111111111111111111111111111")

	caps := common.NewCapabilities()
	caps.Add("symref", "HEAD:/refs/heads/master")
	caps.Add("ofs-delta")
	caps.Add("multi_ack")

	refs := map[string]core.Hash{
		"refs/heads/master": core.NewHash("2222222222222222222222222222222222222222"),
		"refs/tags/v1":      core.NewHash("3333333333333333333333333333333333333333"),
	}

	peeled := map[string]core.Hash{
		"refs/tags/v1": core.NewHash("4444444444444444444444444444444444444444"),
	}

	shallows := []core.Hash{
		core.NewHash("5555555555555555555555555555555555555555"),
	}

	ar := &advrefs.Contents{
		Head:     &head,
		Caps:     caps,
		Refs:     refs,
		Peeled:   peeled,
		Shallows: shallows,
	}
	r, _ := ar.Encode()

	raw, _ := ioutil.ReadAll(r)
	fmt.Printf("%q", string(raw))
	// Output:
	// "00651111111111111111111111111111111111111111 HEAD\x00multi_ack ofs-delta symref=HEAD:/refs/heads/master\n003f2222222222222222222222222222222222222222 refs/heads/master\n003a3333333333333333333333333333333333333333 refs/tags/v1\n003d4444444444444444444444444444444444444444 refs/tags/v1^{}\n0035shallow 5555555555555555555555555555555555555555\n0000"
}
