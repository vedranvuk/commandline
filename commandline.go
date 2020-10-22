// Package commandline implements a command line parser.
package commandline

import (
	"encoding/json"
	"errors"
	"log"
	"reflect"
	"sort"
	"strings"
)

var (
	// ErrInvalidName is returned when an invalid Command or Param long or
	// short name is specified to a registration function.
	ErrInvalidName = errors.New("commandline: invalid name")
	// ErrDuplicateName is returned when an already registered Command or Param
	// long or short name is specified to a registration function.
	ErrDuplicateName = errors.New("commandline: duplicate name")
	// ErrInvalidValue is returned when an invalid parameter is passed as a
	// Param value, i.e. not a pointer to a Go value.
	ErrInvalidValue = errors.New("commandline: invalid value")
	// ErrValueRequired is returned when no Go value is specified for a
	// required Param to a registration function.
	ErrValueRequired = errors.New("commandline: value parameter required")
	// ErrArgumentRequired is returned when no argument is provided on command
	// line to a Param that requires one.
	ErrArgumentRequired = errors.New("commandline: param requires an argument")
)

// CommandFunc is a prototype of a function that handles the event of a
// Command being parsed from command line arguments.
//
// Parser parses Command's Params, pauses parsing and invokes parsed Command's
// CommandFunc carrying Command's Params which can act upon Command invokation
// and decide if parsing should continue depending on the result.
//
// To continue command line parsing the function should return nil.
//
// To stop further command line parsing the function can return an error.
// The error will be propagated back to Parser.Parse caller who is responsible
// for interpreting the error.
//
// See Commands.Register and Params.Register for more details.
type CommandFunc = func(*Params) error

// CommandRawFunc is the raw params version of CommandFunc which passes all
// remaining command line args following the command invocation argument to the
// handler. Registering this handler for a Command disables registering
// sub-Commands for that Command as parsing stops after its invocation.
//
// This is to allow custom argument parsing.
//
// Params can still be defined for a Command having a raw handler using
// RegisterRaw and are addressed by 0-based index. This allows for automatic
// value setting and providing Param naming and help text.
//
// See Commands.Register and Params.RegisterRaw for more details.
type CommandRawFunc = func([]string) error

// Parser is a command line parser. Its' Parse method is to be invoked
// with a slice of command line arguments passed to program.
//
// It is command oriented, meaning that one or more Command instances can be
// defined in Parser's Commands which when parsed from command line arguments
// invoke CommandFuncs registered for those Command instances. Command can have
// its' own Commands so a Command hierarchy can be defined.
//
// Root Commands, as an exception, allow for one Command with an empty name
// to be defined. This is to allow that program args need not start with a
// Command and to allow Params to be passed first which can act as "global".
//
// Command can have one or more Param instances defined in its' Params which
// can be either optional or required, and have both long and short names,
// help text, be marked as required and have a pointer to a Go value assigned
// to them which is written to when Param gets parsed from command line args.
//
// Short Param names have the "-" prefix, can be one character long
// and can be combined together following a short form prefix if none of the
// combined Params require a Param Value.
//
// Long Param names have the "--" prefix and cannot be combined.
//
// If Params are defined as optional they do not cause a parse error if not
// parsed from program args and can have an optional Value following it if a
// target Go value was specified during registration.
// i.e. "-v" or "--verbose" or "-l :8080" or "--listenaddr :8080".
//
// If they are defined as required they cause a parse error if not parsed
// from program args and must have a Value following it.
// i.e. "-u root" or "--password 1337".
//
// See Params.Register for details on how a Param is defined.
//
// Command can have a CommandFunc registered optionaly so that a Command can
// serve solely as sub-Command selector. For more details see CommandFunc.
//
// Command can have a CommandRawFunc registered that causes Parser to parse
// arguments that follow the Command invocation as standalone and indexed in
// order as they appear on the command line. This allows for custom parsers.
//
type Parser struct {
	// Commands is the root command set.
	//
	// Root Commands as an exception allows a single Command
	// with an empty name that serves as "global flag" container.
	Commands
	// args is a slice of arguments being parsed.
	// Args are set once by Parse() then read and updated by Commands
	// and Params down the Parse chain until exhausted or an error occurs.
	args []string
}

// New returns a new instance of *Parser.
func New() *Parser {
	p := &Parser{}
	p.Commands = *newCommands(p)
	return p
}

// Parse parses specified args, usually invoked as "Parse(os.Args[1:])".
// If a parse error occurs or an invoked Command handler returns an error
// it is returned.
func (p *Parser) Parse(args []string) error {
	p.args = args
	return p.Commands.parse(p)
}

// paramdef is a print definition of a param.
type paramdef struct {
	long     string
	short    string
	help     string
	typename string
	reqvalue bool
}

// cmddef is a print definition of a command.
type cmddef struct {
	name   string
	help   string
	indent int
	raw    bool
	params []*paramdef
}

// getPrintDefs recursively constructs sorted print definitions
// from Commands and Params.
func getPrintDefs(cmds *Commands, indent int, items *[]*cmddef) {

	for name, cmd := range cmds.commandmap {
		pc := &cmddef{name: name, help: cmd.help, indent: indent}
		if _, ok := cmd.f.(CommandRawFunc); ok {
			pc.raw = true
		}
		pc.params = make([]*paramdef, 0, len(cmd.Params.longparams))
		for pname, p := range cmd.Params.longparams {
			short := cmd.Params.longtoshort[pname]
			kind := ""
			if p.value != nil {
				kind = reflect.Indirect(reflect.ValueOf(p.value)).Kind().String()
			}
			pc.params = append(pc.params, &paramdef{pname, short, p.help, kind, p.required})
		}
		if !pc.raw {
			sort.Slice(pc.params, func(i, j int) bool { return pc.params[i].long < pc.params[j].long })
		}
		*items = append(*items, pc)
		if len(cmd.commandmap) > 0 {
			getPrintDefs(&cmd.Commands, indent+1, items)
		}
	}
}

// indent constructs an tab indented string of specified depth.
func indent(depth int) string {
	buf := make([]byte, depth*2)
	for i := 0; i < depth; i++ {
		buf[i] = ' '
		buf[i+1] = ' '
	}
	return string(buf)
}

// String implements Stringer on Parser.
func (p *Parser) String() string {

	items := []*cmddef{}
	getPrintDefs(&p.Commands, 0, &items)
	sort.Slice(items, func(i, j int) bool { return items[i].name < items[j].name })

	sb := strings.Builder{}
	for _, cmd := range items {
		sb.WriteString(indent(cmd.indent))
		sb.WriteString(cmd.name)
		sb.WriteRune('\t')
		sb.WriteString(cmd.help)
		sb.WriteRune('\n')
		for _, param := range cmd.params {
			sb.WriteString(indent(cmd.indent))
			sb.WriteRune('\t')
			if param.short != "" && !cmd.raw {
				sb.WriteRune('-')
				sb.WriteString(param.short)
			}
			sb.WriteRune('\t')
			if cmd.raw {
				if param.reqvalue {
					sb.WriteRune('<')
				} else {
					sb.WriteRune('[')
				}
			} else {
				sb.WriteString("--")
			}
			sb.WriteString(param.long)
			if cmd.raw {
				if param.reqvalue {
					sb.WriteRune('>')
				} else {
					sb.WriteRune(']')
				}
			}
			sb.WriteRune('\t')
			if param.reqvalue || cmd.raw {
				sb.WriteString("\t(")
				sb.WriteString(param.typename)
				sb.WriteString(")\t")
			}
			sb.WriteString(param.help)
			sb.WriteRune('\n')
		}
		sb.WriteRune('\n')
	}
	return sb.String()
}

// argKind defines argument kind.
type argKind int

const (
	argNone    argKind = iota // Invalid/no argument.
	argCommand                // Command argument.
	argLong                   // Param with long name.
	argShort                  // Param with short name.
	argComb                   // Combined short Params.
)

// String implements stringer on argKind.
func (ak argKind) String() (s string) {
	switch ak {
	case argNone:
		s = "none"
	case argCommand:
		s = "command"
	case argLong:
		s = "long parameter"
	case argShort:
		s = "short parameter"
	case argComb:
		s = "combined short arguments"
	}
	return
}

// peek returns the first arg in args if args are not empty, otherwise returns
// an empty string.
func (p *Parser) peek() string {
	if len(p.args) > 0 {
		return p.args[0]
	}
	return ""
}

// arg returns the first arg in Parser trimmed of any prefixes and its' kind.
func (p *Parser) arg() (arg string, kind argKind) {
	if len(p.args) == 0 {
		return "", argNone
	}
	arg = p.args[0]
	for i := 0; ; i++ {
		if arg[i] == '-' {
			if kind == argNone {
				kind = argShort
				continue
			}
			if kind == argShort {
				kind = argLong
				continue
			}
		}
		arg = arg[i:]
		if kind == argNone {
			kind = argCommand
		}
		if kind == argShort {
			if len(arg) > 1 {
				kind = argComb
			}
		}
		return
	}
}

// next discards the first arg in the args slice and returns a bool indicating
// if there is any args left.
func (p *Parser) next() bool {
	if len(p.args) == 0 {
		return false
	}
	p.args = p.args[1:]
	return len(p.args) > 0
}

// Command defines a command.
// A Command can contain Commands and propagate Parser args
// further down the Commands chain.
type Command struct {
	help string      // help is the command help text.
	f    interface{} // f is the function to invoke when this COmmand is executed.
	Params
	Commands
}

// commandMap is a map of command name to *Command.
type commandMap map[string]*Command

// Commands holds a set of Commands with a unique name.
type Commands struct {
	// parent is this Commands parent. If it is a Parser
	// this Commands is the root Commands. If it is a Command
	// this is a sub-Command Commands.
	parent interface{}
	// commandmap is a map of command names to *Command definitions.
	commandmap commandMap
}

// newCommands returns a new Commands instance owned by owner.
func newCommands(parent interface{}) *Commands {
	return &Commands{
		parent:     parent,
		commandmap: make(commandMap),
	}
}

// AddCommand registers a new command under specified name and help text that
// invokes f if it is not nil.
//
// If Commands is the root set in a Parser it can register a single
// AddCommand with an empty name which can serve for the purpose of global params.
//
// If a registration error occurs it is returned with a nil AddCommand.
func (c *Commands) AddCommand(name, help string, cmdFunc interface{}) (*Command, error) {

	if name == "" {
		if _, ok := c.parent.(*Parser); !ok {
			return nil, ErrInvalidName
		}
	}

	if _, exists := c.commandmap[name]; exists {
		return nil, ErrDuplicateName
	}

	if parentcmd, ok := c.parent.(*Command); ok {
		if _, ok := parentcmd.f.(CommandRawFunc); ok {
			return nil, errors.New("commandline: cannot register a sub Command in a Command with a CommandRawHandler")
		}
	}

	if cmdFunc != nil {
		if _, ok := cmdFunc.(CommandFunc); !ok {
			if _, ok := cmdFunc.(CommandRawFunc); !ok {
				return nil, errors.New("commandline: invalid cmdFunc parameter")
			}
		}
	}

	cmd := &Command{
		help: help,
		f:    cmdFunc,
	}
	cmd.Params = *newParams(cmd)
	cmd.Commands = *newCommands(cmd)
	c.commandmap[name] = cmd

	return cmd, nil
}

// GetCommand returns a *Command by name if found and truth if found.
func (c *Commands) GetCommand(name string) (cmd *Command, ok bool) {
	cmd, ok = c.commandmap[name]
	return
}

// parse parses Parser args into this Commands.
func (c *Commands) parse(cl *Parser) error {
	arg, kind := cl.arg()
	// Arguments exhausted.
	if kind == argNone {
		return nil
	}
	var cmd *Command
	var exists bool
	var global bool
	if kind != argCommand {
		// Arg is not a Command. See if a special case
		// of single unnamed root command.
		cmd, exists = c.commandmap[""]
		if !exists {
			return errors.New("commandline: expected command, got " + kind.String())
		}
		global = true
	} else {
		// Arg is a Command.
		cmd, exists = c.commandmap[arg]
		if !exists {
			return errors.New("commandline: command '" + arg + "' not found")
		}
	}
	// Advance args.
	if cl.next() {
		// Parse Params.
		if err := cmd.Params.parse(cl, cmd); err != nil {
			return err
		}
		// Early CommandRawFunc invocation passes any
		// remaining args to it and stops further parsing.
		if cmd.f != nil {
			if cmdfunc, ok := cmd.f.(CommandRawFunc); ok {
				return cmdfunc(cl.args)
			}
		}
	}
	// Check if required parameters were parsed.
	for paramname, param := range cmd.Params.longparams {
		if param.required && !param.parsed {
			return errors.New("commandline: required parameter '" + paramname + "' for command '" + arg + "' not specified")
		}
	}
	// Execute Command.
	if cmd.f != nil {
		if cmdfunc, ok := cmd.f.(CommandFunc); ok {
			if err := cmdfunc(&cmd.Params); err != nil {
				return err
			}
		} else {
			panic("commandline: Command.f is not of CommandFunc type")
		}
	}
	if global {
		return c.parse(cl)
	}
	return cmd.Commands.parse(cl)
}

// Param defines a Command parameter contained in a Params.
type Param struct {
	help     string      // help is the Param help text.
	required bool        // required specifies if this Param is required.
	value    interface{} // value is the param value.

	parsed bool // parsed indicates if param was parsed from command line.
}

// parammmap is a map of param name to *Param.
type paramMap map[string]*Param

// longtoshort maps a long param name to short param name.
type longtoshort map[string]string

// Value returns the Param value.
func (p *Param) Value() interface{} { return p.value }

// A Params is a set of Command Params.
type Params struct {
	// cmd is the reference to owner *Command.
	cmd *Command
	// parammap is a map of long param name to *Param.
	shortparams paramMap
	// parammap is a map of short param name to *Param.
	longparams paramMap
	// longtoshort maps a long param name to short param name.
	longtoshort longtoshort
	// rawmap maps a raw param index to a *Param.
	rawparams []string
}

// newParams returns a new instance of *Params.
func newParams(cmd *Command) *Params {
	return &Params{
		cmd,
		make(paramMap),
		make(paramMap),
		make(longtoshort),
		[]string{},
	}
}

// Parsed returns if the param under specified name was parsed.
// If the Param under specified name is not registered, returns false.
func (p *Params) Parsed(name string) bool {
	if param, exists := p.shortparams[name]; exists {
		return param.parsed
	}
	return false
}

// AddParam registers a new AddParam in these Params.
//
// Long param name is required, short is optional and can be empty, as is help.
//
// If required is specified value must be a pointer to a supported Go value
// which will be updated to the value of the AddParam value parsed from args.
// If a required AddParam or its' value is not found in args during this Params
// parsing an ErrValueRequired will be returned.
//
// If AddParam is not marked as required, specifying a pointer to a supported Go
// value via value parameter is optional:
// If nil, a value for the AddParam will not be parsed from args.
// If a pointer to a supported Go value is specified the AddParam when parsed will
// look for an optional AddParam value - and return ErrValueRequired if not found.
//
// Short params that take values, required or optional, cannot be combined.
//
func (p *Params) AddParam(long, short, help string, required bool, value interface{}) error {

	if long == "" {
		return ErrInvalidName
	}

	if _, exists := p.longparams[long]; exists {
		return ErrDuplicateName
	}

	if _, exists := p.shortparams[short]; exists {
		return ErrDuplicateName
	}

	if required && value == nil {
		return ErrValueRequired
	}

	param := &Param{
		help:     help,
		required: required,
		value:    value,
	}

	p.longparams[long] = param
	if short != "" {
		p.shortparams[short] = param
	}
	p.longtoshort[long] = short

	return nil
}

// AddRawParam registers a param for a Command which must have a CommandRawFunc
// handler set. Param has specified name and help and will apply to a raw param
// passed to handler under specified index, which must be in order and unique.
// Only one non-required param is allowed in Params and it must be the last one.
// If value is a pointer to a Go value, value will be set to arg at index of
// this Param as registered.
// If an error occurs it is returned and the Param is not registered.
func (p *Params) AddRawParam(name, help string, index int, required bool, value interface{}) error {

	if _, ok := p.cmd.f.(CommandRawFunc); !ok {
		return errors.New("commandline: cannot register a raw param, command does not have a CommandRawFunc handler")
	}

	if index != len(p.rawparams) {
		return errors.New("commandline: raw param index out of order")
	}

	if len(p.rawparams) > 0 {
		if !p.longparams[p.rawparams[len(p.rawparams)-1]].required {
			if required {
				return errors.New("commandline: cannot add a required parameter after a non-required parameter")
			}
			if !required {
				return errors.New("commandline: cannot add more than one optional parameter")
			}
		}
	}

	if err := p.AddParam(name, "", help, required, value); err != nil {
		return err
	}

	p.rawparams = append(p.rawparams, name)

	return nil
}

// parse parses the Parser args into this Params.
func (p *Params) parse(cl *Parser, cmd *Command) error {
	for {
		arg, kind := cl.arg()
		var param *Param
		var exists bool
		switch kind {
		case argNone:
			return nil
		case argCommand:
			if _, ok := p.cmd.f.(CommandRawFunc); !ok {
				return nil
			}
			for idx, parname := range p.rawparams {
				par := p.longparams[parname]
				if par.value != nil {
					if err := JSONStringToInterface(cl.peek(), par.value); err != nil {
						return err
					}
				}
				par.parsed = true
				if !cl.next() {
					if len(p.rawparams)-1 > idx && par.required {
						return errors.New("commandline: no argument specified for param '" + parname + "'")
					}
					break
				}
			}
			if len(cl.args) > 0 {
				return errors.New("commandline: extra args specified: " + strings.Join(cl.args, " "))
			}
			return nil
		case argShort:
			param, exists = p.shortparams[arg]
			if !exists {
				return errors.New("commandline: short parameter '" + arg + "' not found")
			}
			param.parsed = true
		case argLong:
			param, exists = p.longparams[arg]
			if !exists {
				return errors.New("commandline: long parameter '" + arg + "' not found")
			}
			param.parsed = true
		case argComb:
			shorts := strings.Split(arg, "")
			for _, short := range shorts {
				param, exists = p.shortparams[short]
				if !exists {
					return errors.New("commandline: short parameter '" + short + "' not found")
				}
				if param.Value() != nil {
					return errors.New("commandline: short param '" + short + "' with parameter combined")
				}
				param.parsed = true
			}
		}
		if param.value != nil {
			if !cl.next() {
				return ErrArgumentRequired
			}
			value, _ := cl.arg()
			if err := JSONStringToInterface(value, param.value); err != nil {
				return err
			}
		}
		if !cl.next() {
			return nil
		}
	}
}

// JSONStringToInterface converts a JSON string s to a Go value i which must be
// a value compatible with s or returns an error if one occurs.
func JSONStringToInterface(s string, i interface{}) error {
	v := reflect.Indirect(reflect.ValueOf(i))
	for v.Kind() == reflect.Ptr {
		v = reflect.Indirect(v)
	}
	if v.Kind() == reflect.String {
		s = "\"" + s + "\""
	}
	sr := strings.NewReader(s)
	dec := json.NewDecoder(sr)
	if err := dec.Decode(i); err != nil {
		log.Fatal(err)
	}
	return nil
}
