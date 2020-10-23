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
	// ErrNoArgs is returned by Parse if no arguments were specified on
	// command line and there are defined Commands or Params.
	ErrNoArgs = errors.New("commandline: no arguments")
	// ErrInvalidName is returned by Add* methods when an invalid Command or
	// Param long or short name is specified.
	ErrInvalidName = errors.New("commandline: invalid name")
	// ErrDuplicateName is returned by Add* methods when an already registered
	// Command or Param long or short name is specified.
	ErrDuplicateName = errors.New("commandline: duplicate name")
	// ErrInvalidValue is returned by Add* methods or during parsing if an
	// invalid parameter is given for a Param value, i.e. not a valid pointer
	// to a Go value.
	ErrInvalidValue = errors.New("commandline: invalid value")
	// ErrValueRequired is returned by Add* methods when no Go value is given
	// for a Param marked as required.
	ErrValueRequired = errors.New("commandline: value parameter required")
	// ErrArgumentRequired is returned when no argument is provided on command
	// line for a Param that requires one.
	ErrArgumentRequired = errors.New("commandline: param requires an argument")
)

// CommandFunc is a prototype of a function that handles the event of a
// Command being parsed from command line arguments.
//
// Parser parses Command's Params, pauses parsing on next Command or exhausted
// arguments and invokes parsed Command's CommandFunc carrying parsed Params
// then either continues parsing if the handler returned nil or stops and
// retrurns the error that the handler returned back to the Parse method.
//
// Parse caller is responsible for interpreting that error.
//
// See Commands.AddCommand and Params.AddParam for more details.
//
//
//
// CommandRawFunc is the raw Params version of CommandFunc which passes all
// remaining command line args following the command invocation argument to the
// handler. Registering this handler for a Command disables registering
// sub-Commands for that Command and parsing stops after its' invocation.
//
// This is to allow custom argument parsing.
//
// Params can still be defined for a Command having a raw handler using
// AddRawParam and are mapped on a 0-based index, in order as they are
// registered.
//
// See Commands.AddCommand and Params.AddRawParam for more details.
type CommandFunc = func(*Params) error

// Parser is a command line parser. Its' Parse method is to be invoked
// with a slice of command line arguments passed to program.
// For example:
//  err := Parser.Parse(os.Args[1:])
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
// help text, be required or optional and have a pointer to a Go value assigned
// to them which is written from Param value parsed from command line args.
//
// Short Param names have the "-" prefix, can be one character long and can be
// combined together following the short form prefix if none of the combined
// Params require a Param Value. They are optional per Param.
//
// Long Param names have the "--" prefix and cannot be combined. They are
// required.
//
// If Params are defined as optional they do not cause a parse error if not
// parsed from program args and can be optionally defined to parse a value
// from args if a target Go value was specified during registration.
// i.e. "-v" or "--verbose" or "-l :8080" or "--listenaddr :8080".
//
// If they are defined as required they cause a parse error if not parsed
// from program args and must have a Value following it.
// i.e. "-u root" or "--password 1337".
//
// See Params.AddParam for details on how a Param is defined.
//
// Command can have a CommandFunc registered optionaly so that a Command can
// serve solely as sub-Command selector. For more details see CommandFunc.
//
// Command can have a CommandRawFunc registered that causes Parser to parse
// arguments that follow the Command invocation as standalone and indexed in
// order as they appear on the command line. This allows for custom parsers
// and arrays of arguments.
//
// If no raw Params were defined on a Command with a CommandRawFunc handler
// arguments will not be parsed and will just be passed to the handler as is.
//
type Parser struct {
	// args is a slice of arguments being parsed.
	// Args are set once by Parse() then read and updated by Commands
	// and Params down the Parse chain until exhausted or an error occurs
	// using peek(), arg() and next().
	args []string
	// Commands is the root command set.
	//
	// Root Commands as an exception allows a single Command
	// with an empty name that serves as "global flag" container.
	Commands
}

// New returns a new instance of *Parser.
func New() *Parser {
	p := &Parser{}
	p.Commands = *newCommands(p)
	return p
}

// Parse parses specified args, usually invoked as "Parse(os.Args[1:])".
// If a parse error occurs or an invoked Command handler returns an error
// it is returned. Returns ErrNoArgs if args are empty and there are defined
// Commands or Params.
func (p *Parser) Parse(args []string) error {
	if len(p.Commands.commandmap) > 0 && len(args) == 0 {
		return ErrNoArgs
	}
	p.args = args
	p.reset()
	return p.Commands.parse(p)
}

// reset resets the states prior to parsing.
func (p *Parser) reset() { resetParams(&p.Commands) }

// resetParams recursively resets parsed flag of all Params in Commands.
func resetParams(c *Commands) {
	for _, cmd := range c.commandmap {
		cmd.Params.rawargs = []string{}
		if len(cmd.Params.longparams) > 0 {
			for _, p := range cmd.Params.longparams {
				p.parsed = false
			}
		}
		resetParams(&cmd.Commands)
	}
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

// getPrintDefs recursively prepares print definitions from Commands and Params.
func getPrintDefs(cmds *Commands, indent int, items *[]*cmddef) {

	for name, cmd := range cmds.commandmap {
		pc := &cmddef{name: name, help: cmd.help, indent: indent}
		pc.params = make([]*paramdef, 0, len(cmd.Params.longparams))
		for _, pname := range cmd.Params.longindexes {
			p := cmd.Params.longparams[pname]
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
			if param.reqvalue || (cmd.raw && param.typename != "") {
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
	argNone         argKind = iota // Invalid/no argument.
	argCommandOrRaw                // Command or raw argument.
	argLong                        // Param with long name.
	argShort                       // Param with short name.
	argComb                        // Combined short Params.
)

// String implements stringer on argKind.
func (ak argKind) String() (s string) {
	switch ak {
	case argNone:
		s = "none"
	case argCommandOrRaw:
		s = "command or raw argument"
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
	if len(arg) == 0 {
		return "", argCommandOrRaw
	}
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
			kind = argCommandOrRaw
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

// Command is a command definition.
// A Command can contain sub-Commands and propagate Parser args
// further down the Commands chain. It can have zero or more defined
// Param instances in its' Params.
type Command struct {
	// help is the Command help text.
	help string
	// f is the function to invoke when this Command is executed.
	// Can be nil, CommandFunc or CommandRawFunc.
	f CommandFunc

	Params   // Params are this Command's Params.
	Commands // Commands are this Command's Commands.
}

// newCommand returns a new *Command instance with given help and handler.
func newCommand(help string, f CommandFunc) *Command {
	p := &Command{
		help: help,
		f:    f,
	}
	p.Params = *newParams(p)
	p.Commands = *newCommands(p)
	return p
}

// nameToCommand is a map of command name to *Command.
type nameToCommand map[string]*Command

// Commands holds a set of Commands with a unique name.
type Commands struct {
	// parent is this Commands parent.
	// If it is a Parser this Commands is the root Commands.
	// If it is a Command this is a sub-Command Commands.
	parent interface{}
	// commandmap is a map of command names to *Command definitions.
	commandmap nameToCommand
}

// newCommands returns a new *Commands instance owned by parent.
func newCommands(parent interface{}) *Commands {
	return &Commands{
		parent:     parent,
		commandmap: make(nameToCommand),
	}
}

// AddCommand registers a new Command under specified name and help text that
// invokes f when parsed from arguments if it is not nil. Name is the only
// required parameter and cmdFunc can be of CommandFunc or CommandRawFunc.
//
// If Commands is the root set in a Parser it can register a single Command
// with an empty name to be an unnamed container in a global params pattern.
//
// If a registration error occurs it is returned with a nil *Command.
func (c *Commands) AddCommand(name, help string, f CommandFunc) (*Command, error) {

	// Allow empty Command name only in root.
	if name == "" {
		if _, ok := c.parent.(*Parser); !ok {
			return nil, ErrInvalidName
		}
	}

	// No duplicate names.
	if _, exists := c.commandmap[name]; exists {
		return nil, ErrDuplicateName
	}

	// Disallow adding sub-Commands to a Command that has
	// a CommandRawFunc handler.
	if parentcmd, ok := c.parent.(*Command); ok {
		if parentcmd.Params.hasRawArgs() {
			return nil, errors.New("commandline: cannot register a sub Command in a Command with raw Params")
		}
	}

	// Define and add a new Command to self.
	cmd := newCommand(help, f)
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
	if kind != argCommandOrRaw {
		// Arg is not a Command, but a Param. See if a
		// special case of single unnamed root command.
		cmd, exists = c.commandmap[""]
		if !exists {
			return errors.New("commandline: expected command, got " + kind.String() + " '" + arg + "'")
		}
		global = true
	} else {
		// Arg is a Command.
		cmd, exists = c.commandmap[arg]
		if !exists {
			return errors.New("commandline: command '" + arg + "' not found")
		}
	}

	// Advance to next arg, stop if no more.
	if !global {
		cl.next()
	}

	// Parse Params.
	if err := cmd.Params.parse(cl, cmd); err != nil {
		return err
	}

	// Check if required parameters were parsed.
	for paramname, param := range cmd.Params.longparams {
		if param.required && !param.parsed {
			return errors.New("commandline: required parameter '" + paramname + "' for command '" + arg + "' not specified")
		}
	}

	// Execute Command.
	if cmd.f != nil {
		if err := cmd.f(&cmd.Params); err != nil {
			return err
		}
	}

	// Repeat parse on these Commands if "global params"
	// empty Command name container was invoken.
	if global {
		return c.parse(cl)
	}

	// Or pass control to contained Commands.
	return cmd.Commands.parse(cl)
}

// Param defines a Command parameter contained in a Params.
type Param struct {
	// help is the Param help text.
	help string
	// required specifies if this Param is required.
	required bool
	// value is a pointer to a Go value which which is set
	// from parsed Param value if not nil and  points to a
	// valid target.
	value interface{}
	// raw specifies if this param is a raw param.
	raw bool
	// parsed indicates if Param was parsed from arguments.
	parsed bool
}

// newParam returns a new *Param instance with given help, required and value.
func newParam(help string, required, raw bool, value interface{}) *Param {
	return &Param{
		help:     help,
		required: required,
		raw:      raw,
		value:    value,
	}
}

// nameToParam maps a param name to *Param.
type nameToParam map[string]*Param

// nameToName maps a long param name to short param name.
type nameToName map[string]string

// Value returns the Param value.
func (p *Param) Value() interface{} { return p.value }

// A Params defines a set of Command Params unique by long name.
type Params struct {
	// cmd is the reference to owner *Command.
	cmd *Command
	// longparams is a map of long param name to *Param.
	longparams nameToParam
	// shortparams is a map of short param name to *Param.
	shortparams nameToParam
	// longtoshort maps a long param name to short param name.
	longtoshort nameToName
	// longindexes hold long param names in order as they are added.
	longindexes []string
	// rawargs stores the parsed raw Param instances.
	rawargs []string
}

// newParams returns a new instance of *Params.
func newParams(cmd *Command) *Params {
	return &Params{
		cmd,
		make(nameToParam),
		make(nameToParam),
		make(nameToName),
		[]string{},
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

// RawArgs returnes arguments of raw Params in order as passed on command line.
func (p *Params) RawArgs() []string { return p.rawargs }

// hasRawArgs returns if Params contain one or more raw Param instances.
func (p *Params) hasRawArgs() bool {
	for _, par := range p.longparams {
		if par.raw {
			return true
		}
	}
	return false
}

// addParam is the implementation of AddParam minus the check of adding a
// normal Param to a Command with a CommandRawFunc handler.
func (p *Params) addParam(long, short, help string, required, raw bool, value interface{}) error {

	// Long name must not be empty and short name must be max one char long.
	if long == "" || len(short) > 1 {
		return ErrInvalidName
	}

	// No long duplicates.
	if _, exists := p.longparams[long]; exists {
		return ErrDuplicateName
	}

	// No short duplicates.
	if _, exists := p.shortparams[short]; exists {
		return ErrDuplicateName
	}

	// Continuity checks, if any definitions exist.
	if lp := p.last(); lp != nil {

		if lp.raw {
			if !raw {
				return errors.New("commandline: cannot register a non-raw param after a raw param")
			}
			if !lp.required && !required {
				return errors.New("commandline: cannot add more than one optional parameter")
			}
			if !lp.required && required && !raw {
				return errors.New("commandline: cannot add a required parameter after a non-required parameter")
			}
		}
	}

	// Required params need a valid Go value.
	if value == nil {
		if required {
			return ErrValueRequired
		}
	} else {
		// And require a valid value.
		v := reflect.ValueOf(value)
		if !v.IsValid() || v.Kind() != reflect.Ptr {
			return ErrInvalidValue
		}
	}

	// Add a new param.
	param := newParam(help, required, raw, value)
	p.longparams[long] = param
	if short != "" {
		p.shortparams[short] = param
	}
	p.longtoshort[long] = short
	p.longindexes = append(p.longindexes, long)

	return nil
}

// last returns the last defined arg or nil if none registered.
func (p *Params) last() *Param {
	if len(p.longindexes) == 0 {
		return nil
	}
	return p.longparams[p.longindexes[len(p.longindexes)-1]]
}

// AddParam registers a new Param in these Params.
//
// Long param name is required, short is optional and can be empty, as is help.
//
// If required is specified value must be a pointer to a supported Go value
// which will be updated to the value of the Param value parsed from args.
// If a required Param or its' value is not found in args during this Params
// parsing an ErrValueRequired will be returned.
//
// If Param is not marked as required, specifying a pointer to a supported Go
// value via value parameter is optional:
// If nil, a value for the Param will not be parsed from args.
// If a pointer to a supported Go value the Param when parsed will look for an
// optional Param value - and return ErrValueRequired if not found.
func (p *Params) AddParam(long, short, help string, required bool, value interface{}) error {
	return p.addParam(long, short, help, required, false, value)
}

// AddRawParam registers a Param under specified name which must be unique in
// Params for a Command which must have a CommandRawFunc handler set.
//
// Parsed arguments are applied to raw Params in order as they are defined. If
// Param value is a pointer to a valid Go value argument will be converted to
// that Go value.
//
// A single non-required raw Param is allowed and it must be the last one.
//
// If an error occurs it is returned and the Param is not registered.
func (p *Params) AddRawParam(name, help string, required bool, value interface{}) error {
	return p.addParam(name, "", help, required, true, value)
}

// parse parses the Parser args into this Params.
func (p *Params) parse(cl *Parser, cmd *Command) error {
	i := 0
	for {
		arg, kind := cl.arg()
		var param *Param
		var exists bool
		switch kind {

		case argNone:
			// No arguments left.
			return nil

		case argCommandOrRaw:

			// If no Params were defined abort parsing and store
			// remaining arguments to be available to handler.
			if len(p.longindexes) == 0 {
				p.rawargs = append(p.rawargs, cl.args...)
				return nil
			}

			// Iterated over all defined params and still have an argument.
			if l, lp := len(cmd.Params.longindexes), p.last(); l != 0 && i >= l && lp.raw {
				return errors.New("commandline: extra arguments specified: '" + strings.Join(cl.args, " ") + "'")
			}

			// Only raw params possibly accept non-prefixed arguments.
			// Check if there are any named params left, skip them then
			// try parsing the arg into first raw param.
			// Commands with raw args cannot have sub commands.
			for i < len(p.longindexes) {
				param := cmd.Params.longparams[cmd.Params.longindexes[i]]
				if !param.raw {
					i++
					continue
				}
				break
			}
			if i >= len(p.longindexes) {
				return errors.New("commandline: expected a prefixed param, got raw argument '" + arg + "' or a command")
			}

			// Store remaining arguments and parse them into defined Params.
			p.rawargs = append(p.rawargs, cl.args...)
			for ; i < len(p.longindexes); i++ {
				par := p.longparams[p.longindexes[i]]
				if par.value != nil {
					if err := stringToGoValue(cl.peek(), par.value); err != nil {
						return err
					}
				}
				par.parsed = true
				if !cl.next() {
					if len(p.longindexes)-1 > i && par.required {
						return errors.New("commandline: no argument specified for param '" + p.longindexes[i] + "'")
					}
					break
				}
			}

			// Throw error if extra arguments are passed.
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
			i++

		case argLong:
			param, exists = p.longparams[arg]
			if !exists {
				return errors.New("commandline: long parameter '" + arg + "' not found")
			}
			param.parsed = true
			i++

		case argComb:
			// Parse combined args here and now.
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
				i++
			}

		}

		// Set value of normal Param if required.
		if kind != argComb && param.value != nil {
			if !cl.next() {
				return ErrArgumentRequired
			}
			value, _ := cl.arg()
			if err := stringToGoValue(value, param.value); err != nil {
				return err
			}
		}
		if !cl.next() {
			return nil
		}
		if i >= len(cmd.Params.longindexes) {
			break
		}
	}
	return nil
}

// stringToGoValue converts a string to a Go value or returns an error.
// TODO expand on this.
func stringToGoValue(s string, i interface{}) error {
	return jsonStringToGoValue(s, i)
}

// jsonStringToGoValue converts a json string to a Go value or returns an error.
func jsonStringToGoValue(s string, i interface{}) error {

	// Wrap string s into quotes if target is a string
	// so that unmarshaling succeeds.
	//
	// This needs to be done with object fields as well.
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
