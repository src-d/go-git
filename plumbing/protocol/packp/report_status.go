package packp

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/format/pktline"
)

const (
	ok = "ok"
)

// ReportStatus is a report status message, as used in the git-receive-pack
// process whenever the 'report-status' capability is negotiated.
type ReportStatus struct {
	UnpackStatus    string
	CommandStatuses []*CommandStatus
	withSideband    bool
}

// NewReportStatus creates a new ReportStatus message.
// without support for reading additional withSideband data.
func NewReportStatus() *ReportStatus {
	return &ReportStatus{
		withSideband: false,
	}
}

// NewReportStatusWithSideband creates a new ReportStatus message,
// with support for reading additional withSideband data.
func NewReportStatusWithSideband() *ReportStatus {
	return &ReportStatus{
		withSideband: true,
	}
}

// Error returns the first error if any.
func (s *ReportStatus) Error() error {
	if s.UnpackStatus != ok {
		return fmt.Errorf("unpack error: %s", s.UnpackStatus)
	}

	for _, s := range s.CommandStatuses {
		if err := s.Error(); err != nil {
			return err
		}
	}

	return nil
}

// Encode writes the report status to a writer.
func (s *ReportStatus) Encode(w io.Writer) error {
	e := pktline.NewEncoder(w)
	if err := e.Encodef("unpack %s\n", s.UnpackStatus); err != nil {
		return err
	}

	for _, cs := range s.CommandStatuses {
		if err := cs.encode(w); err != nil {
			return err
		}
	}

	return e.Flush()
}

// Decode reads from the given reader and decodes a report-status message.
func (s *ReportStatus) Decode(r io.Reader) error {
	scan := pktline.NewScanner(r)
	if err := s.scanFirstLine(scan); err != nil {
		return err
	}

	if err := s.decodeReportStatus(scan.Bytes()); err != nil {
		return err
	}

	flushed := false
	for scan.Scan() {
		b := scan.Bytes()
		if isFlush(b) {
			flushed = true

			if s.withSideband {
				continue
			}

			break
		}

		// Only try to decode command status if there hasn't been a flush yet.
		// That means we're still on band 1, the primary payload band (pack status).
		// However, more data may follow on bands 2 or even 3;
		// we should not ignore that data just because band 1 is done.
		if !flushed {
			if err := s.decodeCommandStatus(b); err != nil {
				return err
			}
		}
	}

	if !flushed {
		return fmt.Errorf("missing flush")
	}

	return scan.Err()
}

func (s *ReportStatus) scanFirstLine(scan *pktline.Scanner) error {
	if scan.Scan() {
		return nil
	}

	if scan.Err() != nil {
		return scan.Err()
	}

	return io.ErrUnexpectedEOF
}

func (s *ReportStatus) decodeReportStatus(b []byte) error {
	if isFlush(b) {
		return fmt.Errorf("premature flush")
	}

	b = bytes.TrimSuffix(b, eol)

	line := string(b)
	fields := strings.SplitN(line, " ", 2)
	if len(fields) != 2 || fields[0] != "unpack" {
		return fmt.Errorf("malformed unpack status: %s", line)
	}

	s.UnpackStatus = fields[1]
	return nil
}

func (s *ReportStatus) decodeCommandStatus(b []byte) error {
	b = bytes.TrimSuffix(b, eol)

	line := string(b)
	fields := strings.SplitN(line, " ", 3)
	status := ok
	if len(fields) == 3 && fields[0] == "ng" {
		status = fields[2]
	} else if len(fields) != 2 || fields[0] != "ok" {
		return fmt.Errorf("malformed command status: %s", line)
	}

	cs := &CommandStatus{
		ReferenceName: plumbing.ReferenceName(fields[1]),
		Status:        status,
	}
	s.CommandStatuses = append(s.CommandStatuses, cs)
	return nil
}

// CommandStatus is the status of a reference in a report status.
// See ReportStatus struct.
type CommandStatus struct {
	ReferenceName plumbing.ReferenceName
	Status        string
}

// Error returns the error, if any.
func (s *CommandStatus) Error() error {
	if s.Status == ok {
		return nil
	}

	return fmt.Errorf("command error on %s: %s",
		s.ReferenceName.String(), s.Status)
}

func (s *CommandStatus) encode(w io.Writer) error {
	e := pktline.NewEncoder(w)
	if s.Error() == nil {
		return e.Encodef("ok %s\n", s.ReferenceName.String())
	}

	return e.Encodef("ng %s %s\n", s.ReferenceName.String(), s.Status)
}
