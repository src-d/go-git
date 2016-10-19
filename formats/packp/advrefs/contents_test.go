package advrefs_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"gopkg.in/src-d/go-git.v4/core"
	"gopkg.in/src-d/go-git.v4/formats/packp/advrefs"
	"gopkg.in/src-d/go-git.v4/formats/packp/pktline"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type SuiteAdvRefs struct{}

var _ = Suite(&SuiteAdvRefs{})

func (s *SuiteAdvRefs) TestDecodeEncode(c *C) {
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
		var input io.Reader
		var comment CommentInterface
		{
			r, err := pktline.NewFromStrings(test.input...)
			c.Assert(err, IsNil, Commentf("input = %v\n", test.input))

			var buf bytes.Buffer
			tee := io.TeeReader(r, &buf)

			inputCopy, err := ioutil.ReadAll(tee)
			c.Assert(err, IsNil, Commentf("input = %v\n", test.input))
			comment = Commentf("input = %q\n", string(inputCopy))

			input = &buf
		}

		var expected []byte
		{
			r, err := pktline.NewFromStrings(test.expected...)
			c.Assert(err, IsNil, comment)

			expected, err = ioutil.ReadAll(r)
			c.Assert(err, IsNil, comment)
		}

		var ar *advrefs.AdvRefs
		{
			ar = advrefs.NewAdvRefs()
			d := advrefs.NewDecoder(input)
			err = d.Decode(ar)
			c.Assert(err, IsNil, comment)
		}

		var obtained []byte
		{
			var buf bytes.Buffer
			e := advrefs.NewEncoder(&buf)

			err := e.Encode(ar)
			c.Assert(err, IsNil)

			obtained = buf.Bytes()
		}

		c.Assert(obtained, DeepEquals, expected, comment)
	}
}

func ExampleDecoder_Decode() {
	// Here is a raw advertised-ref message.
	raw := "0065a6930aaee06755d1bdcfd943fbf614e4d92bb0c7 HEAD\x00multi_ack ofs-delta symref=HEAD:/refs/heads/master\n" +
		"003fa6930aaee06755d1bdcfd943fbf614e4d92bb0c7 refs/heads/master\n" +
		"00441111111111111111111111111111111111111111 refs/tags/v2.6.11-tree\n" +
		"00475555555555555555555555555555555555555555 refs/tags/v2.6.11-tree^{}\n" +
		"0035shallow 5dc01c595e6c6ec9ccda4f6f69c131c0dd945f8c\n" +
		"0000"

	// Use the raw message as our input.
	input := strings.NewReader(raw)

	// Create a advref.Decoder reading from our input.
	d := advrefs.NewDecoder(input)

	// Decode the input into a newly allocated AdvRefs value.
	ar := advrefs.NewAdvRefs()
	_ = d.Decode(ar) // error check ignored for brevity

	// Do something interesting with the AdvRefs, e.g. print its contents.
	fmt.Println("head =", ar.Head)
	fmt.Println("caps =", ar.Caps.String())
	fmt.Println("...")
	fmt.Println("shallows =", ar.Shallows)
	// Output: head = a6930aaee06755d1bdcfd943fbf614e4d92bb0c7
	// caps = multi_ack ofs-delta symref=HEAD:/refs/heads/master
	// ...
	// shallows = [5dc01c595e6c6ec9ccda4f6f69c131c0dd945f8c]
}

func ExampleEncoder_Encode() {
	// Create a advrefs.Contents with the contents you want...
	ar := advrefs.NewAdvRefs()

	// ...add a hash for the HEAD...
	head := core.NewHash("1111111111111111111111111111111111111111")
	ar.Head = &head

	// ...add some server capabilities...
	ar.Caps.Add("symref", "HEAD:/refs/heads/master")
	ar.Caps.Add("ofs-delta")
	ar.Caps.Add("multi_ack")

	// ...add a couple of references...
	ar.Refs["refs/heads/master"] = core.NewHash("2222222222222222222222222222222222222222")
	ar.Refs["refs/tags/v1"] = core.NewHash("3333333333333333333333333333333333333333")

	// ...including a peeled ref...
	ar.Peeled["refs/tags/v1"] = core.NewHash("4444444444444444444444444444444444444444")

	// ...and finally add a shallow
	ar.Shallows = append(ar.Shallows, core.NewHash("5555555555555555555555555555555555555555"))

	// Encode the advrefs.Contents to a bytes.Buffer.
	// You can encode into stdout too, but you will miss the `\x00`.
	var buf bytes.Buffer
	e := advrefs.NewEncoder(&buf)
	_ = e.Encode(ar) // error checks ignored for brevity

	// Print the contents of the buffer as a quoted string.
	// Printing is as a non-quoted string will be prettier but you will miss the '\x00'.
	fmt.Printf("%q", buf.String())
	// Output:
	// "00651111111111111111111111111111111111111111 HEAD\x00multi_ack ofs-delta symref=HEAD:/refs/heads/master\n003f2222222222222222222222222222222222222222 refs/heads/master\n003a3333333333333333333333333333333333333333 refs/tags/v1\n003d4444444444444444444444444444444444444444 refs/tags/v1^{}\n0035shallow 5555555555555555555555555555555555555555\n0000"
}
