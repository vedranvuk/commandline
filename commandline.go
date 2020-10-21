package commandline

import (
	"encoding/json"
	"errors"
	"log"
	"reflect"
	"strings"
)

var (
	// ErrInvalidName is returned when an invalid name is specified on
	// registration.
	ErrInvalidName = errors.New("commandline: invalid name")
	// ErrInvalidValue is returned when an invalid parameter is passed as a
	// Param value.
	ErrInvalidValue = errors.New("commandline: invalid value")
	// ErrDuplicateName is returned when an already registered name is
	// specified on registration.
	ErrDuplicateName = errors.New("commandline: duplicate name")
	// ErrValueRequired is returned when a required Param is not parsed from
	// args.
	ErrValueRequired = errors.New("commandline: value parameter required")
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
type CommandFunc func(*Params) error

// Parser is a command line parser. Its' Parse method is to be invoked
// with a slice of command line arguments passed to program.
//
// It is command oriented, meaning that one or more Command instances can be
// defined in Parser's Commands which when parsed from command line arguments
// invoke CommandFuncs registered for those Command instances. Command can have
// its' own Commands so a Command hierarchy can be defined.
//
// Root Commands, as an exception, allows for one Command with an empty name
// to be defined. This is to allow that program args need not start with a
// Command and to allow Params to be passed first which can act as "global".
//
// Command can have one or more Param instances defined in its' Params which
// can be either optional or required, and have both long and short names.
//
// Short Param names have the "-" prefix, can be one character long and can
// be combined together following a single "-" prefix if none of the combined
// Params require or an optional Param Value.
//
// Long Param names have the "--" prefix and cannot be combined.
//
// If Params are defined as optional they do not cause a parse error if not
// parsed from program args and can have an optional Value following it.
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
// Example:
//
//  func cmdListUsers(ps *Params) error {
//      if ps.Parsed("color") {
//		    // Do something colorful.
//      }
//      if ps.Parsed("log") {
//          log.Println("User list requested.")
//      }
//      fmt.Println("No users to list.")
//      return nil
//  }
//
//  cl := New()
//  rootcmd, err := cl.Register("", "Global flags.", nil)
//  rootcmd.Params().Register("verbose", "v", "Be verbose.", false, nil)
//  rootcmd.Params().Register("log", "l", "Log to file.", false, nil)
//  listcmd, err := rootcmd.Register("list", "List various items.", nil)
//  listcmd.Register("users", "List users." cmdListUsers)
//  listcmd.Params().Register("color", "c", "Use colors.", false, &useColors)
//  if err := cl.Parse(os.Args[1:]); err != nil {
//      log.Fatal(err)
//  }
//
// Valid command line for the example would be: "-v list users --color"
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

// Parse parses specified args, usually invoked as Parse(os.Args[1:])
// If a parse error occurs or an invoked CommandFunc returns an error
// it is returned.
func (cl *Parser) Parse(args []string) error {
	cl.args = args
	return cl.Commands.parse(cl)
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
func (cl *Parser) peek() string {
	if len(cl.args) > 0 {
		return cl.args[0]
	}
	return ""
}

// arg returns the first arg in Parser trimmed of any prefixes and its' kind.
func (cl *Parser) arg() (arg string, kind argKind) {
	if len(cl.args) == 0 {
		return "", argNone
	}
	arg = cl.args[0]
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
func (cl *Parser) next() bool {
	cl.args = cl.args[1:]
	return len(cl.args) > 0
}

// Command defines a command.
// A Command can contain Commands and propagate Parser args
// further down the Commands chain.
type Command struct {
	help   string      // help is the command help text.
	f      CommandFunc // f is the function to invoke when this COmmand is executed.
	params *Params
	Commands
}

// Params returns Command's Params.
func (c *Command) Params() *Params { return c.params }

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
func newCommands(owner interface{}) *Commands {
	return &Commands{
		parent:     owner,
		commandmap: make(commandMap),
	}
}

// Register registers a new command under specified name and help text that
// invokes f if it is not nil.
//
// If Commands is the root set in a Parser it can register a single
// Command with an empty name which can serve for the purpose of global params.
//
// If a registration error occurs it is returned with a nil Command.
func (cs *Commands) Register(name, help string, f CommandFunc) (*Command, error) {

	if name == "" {
		if _, ok := cs.parent.(*Parser); !ok {
			return nil, ErrInvalidName
		}
	}

	if _, exists := cs.commandmap[name]; exists {
		return nil, ErrDuplicateName
	}

	cmd := &Command{
		help:     help,
		f:        f,
		params:   newParams(),
		Commands: *newCommands(cs),
	}
	cs.commandmap[name] = cmd

	return cmd, nil
}

// Command returns a *Command by name if found and truth if found.
func (cs *Commands) Command(name string) (cmd *Command, ok bool) {
	cmd, ok = cs.commandmap[name]
	return
}

// parse parses Parser args into this Commands.
func (cs *Commands) parse(cl *Parser) error {
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
		cmd, exists = cs.commandmap[""]
		if !exists {
			return errors.New("commandline: expected command, got " + kind.String())
		}
		global = true
	} else {
		// Arg is a Command.
		cmd, exists = cs.commandmap[arg]
		if !exists {
			return errors.New("commandline: command '" + arg + "' not found")
		}
	}
	// Advance args.
	if cl.next() {
		// Parse Params.
		if err := cmd.params.parse(cl, cmd); err != nil {
			return err
		}
	}
	// Check if required parameters were parsed.
	for paramname, param := range cmd.params.longparams {
		if param.required && !param.parsed {
			return errors.New("commandline: required parameter '" + paramname + "' for command '" + arg + "' not specified")
		}
	}
	// Execute Command.
	if cmd.f != nil {
		if err := cmd.f(cmd.params); err != nil {
			return err
		}
	}
	if global {
		return cs.parse(cl)
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

// Value returns the Param value.
func (p *Param) Value() interface{} { return p.value }

// A Params is a set of Command Params.
type Params struct {
	// parammap is a map of long param name to *Param.
	shortparams paramMap
	// parammap is a map of short param name to *Param.
	longparams paramMap
}

// newParams returns a new instance of *Params.
func newParams() *Params {
	return &Params{
		make(paramMap),
		make(paramMap),
	}
}

// Register registers a new Param in these Params.
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
// If a pointer to a supported Go value is specified the Param when parsed will
// look for an optional Param value - and return ErrValueRequired if not found.
//
// Short params that take values, required or optional, cannot be combined.
//
func (ps *Params) Register(long, short, help string, required bool, value interface{}) error {

	if long == "" {
		return ErrInvalidName
	}

	if _, exists := ps.longparams[long]; exists {
		return ErrDuplicateName
	}

	if _, exists := ps.shortparams[short]; exists {
		return ErrDuplicateName
	}

	if required && value == nil {
		return ErrValueRequired
	}

	p := &Param{
		help:     help,
		required: required,
		value:    value,
	}

	ps.longparams[long] = p
	ps.shortparams[short] = p

	return nil
}

// Parsed returns if the param under specified name was parsed.
// If the Param under specified name is not registered, returns false.
func (ps *Params) Parsed(name string) bool {
	if param, exists := ps.shortparams[name]; exists {
		return param.parsed
	}
	return false
}

// parse parses the Parser args into this Params.
func (ps *Params) parse(cl *Parser, cmd *Command) error {
	for {
		arg, kind := cl.arg()
		var param *Param
		var exists bool
		switch kind {
		case argNone, argCommand:
			return nil
		case argShort:
			param, exists = ps.shortparams[arg]
			if !exists {
				return errors.New("commandline: short parameter '" + arg + "' not found")
			}
			param.parsed = true
		case argLong:
			param, exists = ps.longparams[arg]
			if !exists {
				return errors.New("commandline: long parameter '" + arg + "' not found")
			}
			param.parsed = true
		case argComb:
			shorts := strings.Split(arg, "")
			for _, short := range shorts {
				param, exists = ps.shortparams[short]
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
				return ErrValueRequired
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
