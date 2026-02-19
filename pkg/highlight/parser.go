package highlight

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/dlclark/regexp2"
	"gopkg.in/yaml.v2"
)

// A Group represents a syntax group
type Group uint8

// Groups contains all of the groups that are defined
// You can access them in the map via their string name
var Groups map[string]Group
var numGroups Group

// String returns the group name attached to the specific group
func (g Group) String() string {
	for k, v := range Groups {
		if v == g {
			return k
		}
	}
	return ""
}

// A Def is a full syntax definition for a language
// It has a filetype, information about how to detect the filetype based
// on filename or header (the first line of the file)
// Then it has the rules which define how to highlight the file
type Def struct {
	*Header
	rules *rules
}

type Header struct {
	FileType       string
	FileNameRegex  *regexp2.Regexp
	HeaderRegex    *regexp2.Regexp
	SignatureRegex *regexp2.Regexp
}

type HeaderYaml struct {
	FileType string `yaml:"filetype"`
	Detect   struct {
		FNameRegexStr     string `yaml:"filename"`
		HeaderRegexStr    string `yaml:"header"`
		SignatureRegexStr string `yaml:"signature"`
	} `yaml:"detect"`
}

type File struct {
	FileType string
	yamlSrc  map[any]any
}

// A Pattern is one simple syntax rule
// It has a group that the rule belongs to, as well as
// the regular expression to match the pattern
type pattern struct {
	group Group
	regex *regexp2.Regexp
}

// rules defines which patterns and regions can be used to highlight
// a filetype
type rules struct {
	regions  []*region
	patterns []*pattern
	includes []string
}

// A region is a highlighted region (such as a multiline comment, or a string)
// It belongs to a group, and has start and end regular expressions
// A region also has rules of its own that only apply when matching inside the
// region and also rules from the above region do not match inside this region
// Note that a region may contain more regions
type region struct {
	group      Group
	limitGroup Group
	parent     *region
	start      *regexp2.Regexp
	end        *regexp2.Regexp
	skip       *regexp2.Regexp
	rules      *rules
}

func init() {
	Groups = make(map[string]Group)
}

// MakeHeader takes a header (.hdr file) file and parses the header
// Header files make parsing more efficient when you only want to compute
// on the headers of syntax files
// A yaml file might take ~400us to parse while a header file only takes ~20us
func MakeHeader(data []byte) (*Header, error) {
	lines := bytes.Split(data, []byte{'\n'})
	if len(lines) < 4 {
		return nil, errors.New("Header file has incorrect format")
	}
	header := new(Header)
	var err error
	header.FileType = string(lines[0])
	fnameRegexStr := string(lines[1])
	headerRegexStr := string(lines[2])
	signatureRegexStr := string(lines[3])

	if fnameRegexStr != "" {
		header.FileNameRegex, err = compileRegex(fnameRegexStr)
	}
	if err == nil && headerRegexStr != "" {
		header.HeaderRegex, err = compileRegex(headerRegexStr)
	}
	if err == nil && signatureRegexStr != "" {
		header.SignatureRegex, err = compileRegex(signatureRegexStr)
	}

	if err != nil {
		return nil, err
	}

	return header, nil
}

// MakeHeaderYaml takes a yaml spec for a syntax file and parses the
// header
func MakeHeaderYaml(data []byte) (*Header, error) {
	var hdrYaml HeaderYaml
	err := yaml.Unmarshal(data, &hdrYaml)
	if err != nil {
		return nil, err
	}

	header := new(Header)
	header.FileType = hdrYaml.FileType

	if hdrYaml.Detect.FNameRegexStr != "" {
		header.FileNameRegex, err = compileRegex(hdrYaml.Detect.FNameRegexStr)
	}
	if err == nil && hdrYaml.Detect.HeaderRegexStr != "" {
		header.HeaderRegex, err = compileRegex(hdrYaml.Detect.HeaderRegexStr)
	}
	if err == nil && hdrYaml.Detect.SignatureRegexStr != "" {
		header.SignatureRegex, err = compileRegex(hdrYaml.Detect.SignatureRegexStr)
	}

	if err != nil {
		return nil, err
	}

	return header, nil
}

// MatchFileName will check the given file name with the stored regex
func (header *Header) MatchFileName(filename string) bool {
	if header.FileNameRegex != nil {
		return matchString(header.FileNameRegex, filename)
	}

	return false
}

func (header *Header) MatchFileHeader(firstLine []byte) bool {
	if header.HeaderRegex != nil {
		return matchBytes(header.HeaderRegex, firstLine)
	}

	return false
}

// HasFileSignature checks the presence of a stored signature
func (header *Header) HasFileSignature() bool {
	return header.SignatureRegex != nil
}

// MatchFileSignature will check the given line with the stored regex
func (header *Header) MatchFileSignature(line []byte) bool {
	if header.SignatureRegex != nil {
		return matchBytes(header.SignatureRegex, line)
	}

	return false
}

func ParseFile(input []byte) (f *File, err error) {
	// This is just so if we have an error, we can exit cleanly and return the parse error to the user
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
			}
		}
	}()

	var rules map[any]any
	if err = yaml.Unmarshal(input, &rules); err != nil {
		return nil, err
	}

	f = new(File)
	f.yamlSrc = rules

	for k, v := range rules {
		if k == "filetype" {
			filetype := v.(string)

			if filetype == "" {
				return nil, errors.New("empty filetype")
			}

			f.FileType = filetype
			break
		}
	}

	if f.FileType == "" {
		return nil, errors.New("missing filetype")
	}

	return f, err
}

// ParseDef parses an input syntax file into a highlight Def
func ParseDef(f *File, header *Header) (s *Def, err error) {
	// This is just so if we have an error, we can exit cleanly and return the parse error to the user
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
			}
		}
	}()

	src := f.yamlSrc

	s = new(Def)
	s.Header = header

	for k, v := range src {
		if k == "rules" {
			inputRules := v.([]any)

			rules, err := parseRules(inputRules, nil)
			if err != nil {
				return nil, err
			}

			s.rules = rules
		}
	}

	if s.rules == nil {
		// allow empty rules
		s.rules = &rules{}
	}

	return s, err
}

// HasIncludes returns whether this syntax def has any include statements
func HasIncludes(d *Def) bool {
	hasIncludes := len(d.rules.includes) > 0
	for _, r := range d.rules.regions {
		hasIncludes = hasIncludes || hasIncludesInRegion(r)
	}
	return hasIncludes
}

func hasIncludesInRegion(region *region) bool {
	hasIncludes := len(region.rules.includes) > 0
	for _, r := range region.rules.regions {
		hasIncludes = hasIncludes || hasIncludesInRegion(r)
	}
	return hasIncludes
}

// GetIncludes returns a list of filetypes that are included by this syntax def
func GetIncludes(d *Def) []string {
	includes := d.rules.includes
	for _, r := range d.rules.regions {
		includes = append(includes, getIncludesInRegion(r)...)
	}
	return includes
}

func getIncludesInRegion(region *region) []string {
	includes := region.rules.includes
	for _, r := range region.rules.regions {
		includes = append(includes, getIncludesInRegion(r)...)
	}
	return includes
}

// ResolveIncludes will sort out the rules for including other filetypes
// You should call this after parsing all the Defs
func ResolveIncludes(def *Def, files []*File) {
	resolveIncludesInDef(files, def)
}

func resolveIncludesInDef(files []*File, d *Def) {
	for _, lang := range d.rules.includes {
		for _, searchFile := range files {
			if lang == searchFile.FileType {
				searchDef, _ := ParseDef(searchFile, nil)
				d.rules.patterns = append(d.rules.patterns, searchDef.rules.patterns...)
				d.rules.regions = append(d.rules.regions, searchDef.rules.regions...)
			}
		}
	}
	for _, r := range d.rules.regions {
		resolveIncludesInRegion(files, r)
		r.parent = nil
	}
}

func resolveIncludesInRegion(files []*File, region *region) {
	for _, lang := range region.rules.includes {
		for _, searchFile := range files {
			if lang == searchFile.FileType {
				searchDef, _ := ParseDef(searchFile, nil)
				region.rules.patterns = append(region.rules.patterns, searchDef.rules.patterns...)
				region.rules.regions = append(region.rules.regions, searchDef.rules.regions...)
			}
		}
	}
	for _, r := range region.rules.regions {
		resolveIncludesInRegion(files, r)
		r.parent = region
	}
}

func parseRules(input []any, curRegion *region) (ru *rules, err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
			}
		}
	}()
	ru = new(rules)

	for _, v := range input {
		rule := v.(map[any]any)
		for k, val := range rule {
			group := k

			switch object := val.(type) {
			case string:
				if object == "" {
					return nil, fmt.Errorf("Empty rule %s", k)
				}

				if k == "include" {
					ru.includes = append(ru.includes, object)
				} else {
					// Pattern
					r, err := compileRegex(object)
					if err != nil {
						return nil, err
					}

					groupStr := group.(string)
					if _, ok := Groups[groupStr]; !ok {
						numGroups++
						Groups[groupStr] = numGroups
					}
					groupNum := Groups[groupStr]
					ru.patterns = append(ru.patterns, &pattern{groupNum, r})
				}
			case map[any]any:
				// region
				region, err := parseRegion(group.(string), object, curRegion)
				if err != nil {
					return nil, err
				}
				ru.regions = append(ru.regions, region)
			default:
				return nil, fmt.Errorf("Bad type %T", object)
			}
		}
	}

	return ru, nil
}

func parseRegion(group string, regionInfo map[any]any, prevRegion *region) (r *region, err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
			}
		}
	}()

	r = new(region)
	if _, ok := Groups[group]; !ok {
		numGroups++
		Groups[group] = numGroups
	}
	groupNum := Groups[group]
	r.group = groupNum
	r.parent = prevRegion

	// start is mandatory
	if start, ok := regionInfo["start"]; ok {
		start := start.(string)
		if start == "" {
			return nil, fmt.Errorf("Empty start in %s", group)
		}

		r.start, err = compileRegex(start)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("Missing start in %s", group)
	}

	// end is mandatory
	if end, ok := regionInfo["end"]; ok {
		end := end.(string)
		if end == "" {
			return nil, fmt.Errorf("Empty end in %s", group)
		}

		r.end, err = compileRegex(end)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("Missing end in %s", group)
	}

	// skip is optional
	if skip, ok := regionInfo["skip"]; ok {
		skip := skip.(string)
		if skip == "" {
			return nil, fmt.Errorf("Empty skip in %s", group)
		}

		r.skip, err = compileRegex(skip)
		if err != nil {
			return nil, err
		}
	}

	// limit-color is optional
	if groupStr, ok := regionInfo["limit-group"]; ok {
		groupStr := groupStr.(string)
		if groupStr == "" {
			return nil, fmt.Errorf("Empty limit-group in %s", group)
		}

		if _, ok := Groups[groupStr]; !ok {
			numGroups++
			Groups[groupStr] = numGroups
		}
		groupNum := Groups[groupStr]
		r.limitGroup = groupNum

		if err != nil {
			return nil, err
		}
	} else {
		r.limitGroup = r.group
	}

	// rules are optional
	if rules, ok := regionInfo["rules"]; ok {
		r.rules, err = parseRules(rules.([]any), r)
		if err != nil {
			return nil, err
		}
	}

	if r.rules == nil {
		// allow empty rules
		r.rules = &rules{}
	}

	return r, nil
}
