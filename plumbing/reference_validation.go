package plumbing

import (
	"errors"
	"regexp"
	"strings"
)

type ActionChoice int

const (
	// This skips the check
	SKIP ActionChoice = iota
	// This removes the issue when possible
	SANITIZE
	// This throws an error and must be fixed
	VALIDATE
)

var (
	ErrRefLeadingDot                = errors.New("ref name cannot begin with a dot")
	ErrRefTrailingLock              = errors.New("ref name cannot end with .lock")
	ErrRefAtLeastOneForwardSlash    = errors.New("ref name must have at least one forward slash")
	ErrRefDoubleDots                = errors.New("ref name cannot include two consecutive dots")
	ErrRefExcludedCharacters        = errors.New("ref name cannot include many special characters")
	ErrRefLeadingForwardSlash       = errors.New("ref name cannot start with a forward slash")
	ErrRefTrailingForwardSlash      = errors.New("ref name cannot end with a forward slash")
	ErrRefConsecutiveForwardSlashes = errors.New("ref name cannot have consectutive forward slashes")
	ErrRefTrailingDot               = errors.New("ref name cannot end with a dot")
	ErrRefAtOpenBrace               = errors.New("ref name cannot include at-open-brace")
)

var (
	PatternLeadingDot                  = regexp.MustCompile(`^\.`)
	PatternTrailingLock                = regexp.MustCompile(`\.lock$`)
	PatternAtLeastOneForwardSlash      = regexp.MustCompile(`^[^/]+$`)
	PatternDoubleDots                  = regexp.MustCompile(`\.\.`)
	PatternExcludedCharacters          = regexp.MustCompile(`[\000-\037\177 ~^:?*[]+`)
	PatternLeadingForwardSlash         = regexp.MustCompile(`^/`)
	PatternTrailingForwardSlash        = regexp.MustCompile(`/$`)
	PatternConsecutiveForwardSlashes   = regexp.MustCompile(`//+`)
	PatternTrailingDot                 = regexp.MustCompile(`\.$`)
	PatternAtOpenBrace                 = regexp.MustCompile(`@{`)
	PatternExcludedCharactersAlternate = regexp.MustCompile(`[\000-\037\177 ~^:?[]+`)
	PatternOneAllowedAsterisk          = regexp.MustCompile(`^[^*]+?\*?[^*]+?$`)
)

type CheckRefOptions struct {
	// They must contain at least one /
	//  If the --allow-onelevel option is used, this rule is waived.
	AllowOneLevel bool
	// If this option is enabled, <refname> is allowed to contain a
	// single * in the refspec
	RefSpecPattern bool
	// Normalize refname by removing any leading slash (/) characters and
	// collapsing runs of adjacent slashes between name components into
	// a single slash.
	Normalize bool
}

type ActionOptions struct {

	// no slash-separated component can begin with a dot .
	HandleLeadingDot ActionChoice

	// no slash-separated component can end with the sequence .lock
	HandleTrailingLock ActionChoice
	// They must contain at least one /.
	HandleAtLeastOneForwardSlash ActionChoice
	// They cannot have two consecutive dots .. anywhere.
	HandleDoubleDots ActionChoice
	// They cannot have ASCII control characters (i.e. bytes whose values
	// are lower than \040, or \177 DEL), space, tilde ~, caret ^, or
	// colon : anywhere.
	// They cannot have question-mark ?, asterisk *, or open
	// bracket [ anywhere
	// They cannot contain a \
	HandleExcludedCharacters ActionChoice
	// They cannot begin or end with a slash /
	HandleLeadingForwardSlash ActionChoice
	// They cannot begin or end with a slash / or contain
	// multiple consecutive slashes
	HandleTrailingForwardSlash ActionChoice
	// They cannot  contain multiple consecutive slashes
	HandleConsecutiveForwardSlashes ActionChoice
	// They cannot end with a dot .
	HandleTrailingDot ActionChoice
	// They cannot contain a sequence @{
	HandleAtOpenBrace ActionChoice
}

// https://git-scm.com/docs/git-check-ref-format
// git-check-ref-format
type RefNameChecker struct {
	Name            ReferenceName
	CheckRefOptions CheckRefOptions
	ActionOptions   ActionOptions
}

func NewCheckRefOptions(default_value bool) *CheckRefOptions {
	return &CheckRefOptions{
		AllowOneLevel:  default_value,
		RefSpecPattern: default_value,
		Normalize:      default_value,
	}
}

func NewActionOptions(default_value ActionChoice) *ActionOptions {
	return &ActionOptions{
		HandleLeadingDot:                default_value,
		HandleTrailingLock:              default_value,
		HandleAtLeastOneForwardSlash:    default_value,
		HandleDoubleDots:                default_value,
		HandleExcludedCharacters:        default_value,
		HandleLeadingForwardSlash:       default_value,
		HandleTrailingForwardSlash:      default_value,
		HandleConsecutiveForwardSlashes: default_value,
		HandleTrailingDot:               default_value,
		HandleAtOpenBrace:               default_value,
	}
}

func NewRefNameChecker(
	name ReferenceName,
	ref_options CheckRefOptions,
	action_options ActionOptions,
) *RefNameChecker {
	return &RefNameChecker{
		Name:            name,
		CheckRefOptions: ref_options,
		ActionOptions:   action_options,
	}
}

func (v *RefNameChecker) HandleLeadingDot() error {
	switch v.ActionOptions.HandleLeadingDot {
	case VALIDATE:
		if PatternLeadingDot.MatchString(v.Name.String()) {
			return ErrRefLeadingDot
		}
		break
	case SANITIZE:
		v.Name = ReferenceName(PatternLeadingDot.ReplaceAllString(v.Name.String(), ""))
	}
	return nil
}

func (v *RefNameChecker) HandleTrailingLock() error {
	switch v.ActionOptions.HandleTrailingLock {
	case VALIDATE:
		if PatternTrailingLock.MatchString(v.Name.String()) {
			return ErrRefTrailingLock
		}
		break
	case SANITIZE:
		v.Name = ReferenceName(PatternTrailingLock.ReplaceAllString(v.Name.String(), ""))
	}
	return nil
}

func (v *RefNameChecker) HandleAtLeastOneForwardSlash() error {
	if SKIP == v.ActionOptions.HandleAtLeastOneForwardSlash {
		return nil
	}
	count := strings.Count(v.Name.String(), "/")
	if 1 > count {
		if v.CheckRefOptions.AllowOneLevel {
			return nil
		}
		return ErrRefAtLeastOneForwardSlash
	}
	return nil
}

func (v *RefNameChecker) HandleDoubleDots() error {
	switch v.ActionOptions.HandleDoubleDots {
	case VALIDATE:
		if PatternDoubleDots.MatchString(v.Name.String()) {
			return ErrRefDoubleDots
		}
		break
	case SANITIZE:
		v.Name = ReferenceName(PatternDoubleDots.ReplaceAllString(v.Name.String(), ""))
	}
	return nil
}

func (v *RefNameChecker) HandleExcludedCharacters() error {
	switch v.ActionOptions.HandleExcludedCharacters {
	case VALIDATE:
		if PatternExcludedCharacters.MatchString(v.Name.String()) {
			return ErrRefExcludedCharacters
		}
		break
	case SANITIZE:
		if v.CheckRefOptions.RefSpecPattern && PatternOneAllowedAsterisk.MatchString(v.Name.String()) {
			v.Name = ReferenceName(PatternExcludedCharactersAlternate.ReplaceAllString(v.Name.String(), ""))

		} else {
			v.Name = ReferenceName(PatternExcludedCharacters.ReplaceAllString(v.Name.String(), ""))
		}
	}
	return nil
}

func (v *RefNameChecker) HandleLeadingForwardSlash() error {
	switch v.ActionOptions.HandleLeadingForwardSlash {
	case VALIDATE:
		if PatternLeadingForwardSlash.MatchString(v.Name.String()) {
			return ErrRefLeadingForwardSlash
		}
		break
	case SANITIZE:
		v.Name = ReferenceName(PatternLeadingForwardSlash.ReplaceAllString(v.Name.String(), ""))
	}
	return nil
}

func (v *RefNameChecker) HandleTrailingForwardSlash() error {
	switch v.ActionOptions.HandleTrailingForwardSlash {
	case VALIDATE:
		if PatternTrailingForwardSlash.MatchString(v.Name.String()) {
			return ErrRefTrailingForwardSlash
		}
		break
	case SANITIZE:
		v.Name = ReferenceName(PatternTrailingForwardSlash.ReplaceAllString(v.Name.String(), ""))
	}
	return nil
}

func (v *RefNameChecker) HandleConsecutiveForwardSlashes() error {
	if SKIP == v.ActionOptions.HandleConsecutiveForwardSlashes {
		return nil
	}
	if PatternConsecutiveForwardSlashes.MatchString(v.Name.String()) {
		if SANITIZE == v.ActionOptions.HandleConsecutiveForwardSlashes {
			if v.CheckRefOptions.Normalize {
				v.Name = ReferenceName(PatternConsecutiveForwardSlashes.ReplaceAllString(v.Name.String(), "/"))
				return nil
			}
		}
		return ErrRefConsecutiveForwardSlashes
	}
	return nil
}

func (v *RefNameChecker) HandleTrailingDot() error {
	switch v.ActionOptions.HandleTrailingDot {
	case VALIDATE:
		if PatternTrailingDot.MatchString(v.Name.String()) {
			return ErrRefTrailingDot
		}
		break
	case SANITIZE:
		v.Name = ReferenceName(PatternTrailingDot.ReplaceAllString(v.Name.String(), ""))
	}
	return nil
}

func (v *RefNameChecker) HandleAtOpenBrace() error {
	switch v.ActionOptions.HandleAtOpenBrace {
	case VALIDATE:
		if PatternAtOpenBrace.MatchString(v.Name.String()) {
			return ErrRefAtOpenBrace
		}
		break
	case SANITIZE:
		v.Name = ReferenceName(PatternAtOpenBrace.ReplaceAllString(v.Name.String(), ""))
	}
	return nil
}

func (v *RefNameChecker) CheckRefName() error {
	handles := []func() error{
		v.HandleLeadingDot,
		v.HandleTrailingLock,
		v.HandleAtLeastOneForwardSlash,
		v.HandleDoubleDots,
		v.HandleExcludedCharacters,
		v.HandleLeadingForwardSlash,
		v.HandleTrailingForwardSlash,
		v.HandleConsecutiveForwardSlashes,
		v.HandleTrailingDot,
		v.HandleAtOpenBrace,
	}
	for _, handle := range handles {
		err := handle()
		if nil != err {
			return err
		}
	}
	return nil
}
