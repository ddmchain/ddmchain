package cli

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"time"
)

var (
	changeLogURL                    = "https://github.com/urfave/cli/blob/master/CHANGELOG.md"
	appActionDeprecationURL         = fmt.Sprintf("%s#deprecated-cli-app-action-signature", changeLogURL)
	runAndExitOnErrorDeprecationURL = fmt.Sprintf("%s#deprecated-cli-app-runandexitonerror", changeLogURL)

	contactSysadmin = "This is an error in the application.  Please contact the distributor of this application if this is not you."

	errInvalidActionType = NewExitError("ERROR invalid Action type. "+
		fmt.Sprintf("Must be `func(*Context`)` or `func(*Context) error).  %s", contactSysadmin)+
		fmt.Sprintf("See %s", appActionDeprecationURL), 2)
)

type App struct {

	Name string

	HelpName string

	Usage string

	UsageText string

	ArgsUsage string

	Version string

	Description string

	Commands []Command

	Flags []Flag

	EnableBashCompletion bool

	HideHelp bool

	HideVersion bool

	categories CommandCategories

	BashComplete BashCompleteFunc

	Before BeforeFunc

	After AfterFunc

	Action interface{}

	CommandNotFound CommandNotFoundFunc

	OnUsageError OnUsageErrorFunc

	Compiled time.Time

	Authors []Author

	Copyright string

	Author string

	Email string

	Writer io.Writer

	ErrWriter io.Writer

	Metadata map[string]interface{}

	ExtraInfo func() map[string]string

	CustomAppHelpTemplate string

	didSetup bool
}

func compileTime() time.Time {
	info, err := os.Stat(os.Args[0])
	if err != nil {
		return time.Now()
	}
	return info.ModTime()
}

func NewApp() *App {
	return &App{
		Name:         filepath.Base(os.Args[0]),
		HelpName:     filepath.Base(os.Args[0]),
		Usage:        "A new cli application",
		UsageText:    "",
		Version:      "0.0.0",
		BashComplete: DefaultAppComplete,
		Action:       helpCommand.Action,
		Compiled:     compileTime(),
		Writer:       os.Stdout,
	}
}

func (a *App) Setup() {
	if a.didSetup {
		return
	}

	a.didSetup = true

	if a.Author != "" || a.Email != "" {
		a.Authors = append(a.Authors, Author{Name: a.Author, Email: a.Email})
	}

	newCmds := []Command{}
	for _, c := range a.Commands {
		if c.HelpName == "" {
			c.HelpName = fmt.Sprintf("%s %s", a.HelpName, c.Name)
		}
		newCmds = append(newCmds, c)
	}
	a.Commands = newCmds

	if a.Command(helpCommand.Name) == nil && !a.HideHelp {
		a.Commands = append(a.Commands, helpCommand)
		if (HelpFlag != BoolFlag{}) {
			a.appendFlag(HelpFlag)
		}
	}

	if !a.HideVersion {
		a.appendFlag(VersionFlag)
	}

	a.categories = CommandCategories{}
	for _, command := range a.Commands {
		a.categories = a.categories.AddCommand(command.Category, command)
	}
	sort.Sort(a.categories)

	if a.Metadata == nil {
		a.Metadata = make(map[string]interface{})
	}

	if a.Writer == nil {
		a.Writer = os.Stdout
	}
}

func (a *App) Run(arguments []string) (err error) {
	a.Setup()

	shellComplete, arguments := checkShellCompleteFlag(a, arguments)

	set, err := flagSet(a.Name, a.Flags)
	if err != nil {
		return err
	}

	set.SetOutput(ioutil.Discard)
	err = set.Parse(arguments[1:])
	nerr := normalizeFlags(a.Flags, set)
	context := NewContext(a, set, nil)
	if nerr != nil {
		fmt.Fprintln(a.Writer, nerr)
		ShowAppHelp(context)
		return nerr
	}
	context.shellComplete = shellComplete

	if checkCompletions(context) {
		return nil
	}

	if err != nil {
		if a.OnUsageError != nil {
			err := a.OnUsageError(context, err, false)
			HandleExitCoder(err)
			return err
		}
		fmt.Fprintf(a.Writer, "%s %s\n\n", "Incorrect Usage.", err.Error())
		ShowAppHelp(context)
		return err
	}

	if !a.HideHelp && checkHelp(context) {
		ShowAppHelp(context)
		return nil
	}

	if !a.HideVersion && checkVersion(context) {
		ShowVersion(context)
		return nil
	}

	if a.After != nil {
		defer func() {
			if afterErr := a.After(context); afterErr != nil {
				if err != nil {
					err = NewMultiError(err, afterErr)
				} else {
					err = afterErr
				}
			}
		}()
	}

	if a.Before != nil {
		beforeErr := a.Before(context)
		if beforeErr != nil {
			ShowAppHelp(context)
			HandleExitCoder(beforeErr)
			err = beforeErr
			return err
		}
	}

	args := context.Args()
	if args.Present() {
		name := args.First()
		c := a.Command(name)
		if c != nil {
			return c.Run(context)
		}
	}

	if a.Action == nil {
		a.Action = helpCommand.Action
	}

	err = HandleAction(a.Action, context)

	HandleExitCoder(err)
	return err
}

func (a *App) RunAndExitOnError() {
	if err := a.Run(os.Args); err != nil {
		fmt.Fprintln(a.errWriter(), err)
		OsExiter(1)
	}
}

func (a *App) RunAsSubcommand(ctx *Context) (err error) {

	if len(a.Commands) > 0 {
		if a.Command(helpCommand.Name) == nil && !a.HideHelp {
			a.Commands = append(a.Commands, helpCommand)
			if (HelpFlag != BoolFlag{}) {
				a.appendFlag(HelpFlag)
			}
		}
	}

	newCmds := []Command{}
	for _, c := range a.Commands {
		if c.HelpName == "" {
			c.HelpName = fmt.Sprintf("%s %s", a.HelpName, c.Name)
		}
		newCmds = append(newCmds, c)
	}
	a.Commands = newCmds

	set, err := flagSet(a.Name, a.Flags)
	if err != nil {
		return err
	}

	set.SetOutput(ioutil.Discard)
	err = set.Parse(ctx.Args().Tail())
	nerr := normalizeFlags(a.Flags, set)
	context := NewContext(a, set, ctx)

	if nerr != nil {
		fmt.Fprintln(a.Writer, nerr)
		fmt.Fprintln(a.Writer)
		if len(a.Commands) > 0 {
			ShowSubcommandHelp(context)
		} else {
			ShowCommandHelp(ctx, context.Args().First())
		}
		return nerr
	}

	if checkCompletions(context) {
		return nil
	}

	if err != nil {
		if a.OnUsageError != nil {
			err = a.OnUsageError(context, err, true)
			HandleExitCoder(err)
			return err
		}
		fmt.Fprintf(a.Writer, "%s %s\n\n", "Incorrect Usage.", err.Error())
		ShowSubcommandHelp(context)
		return err
	}

	if len(a.Commands) > 0 {
		if checkSubcommandHelp(context) {
			return nil
		}
	} else {
		if checkCommandHelp(ctx, context.Args().First()) {
			return nil
		}
	}

	if a.After != nil {
		defer func() {
			afterErr := a.After(context)
			if afterErr != nil {
				HandleExitCoder(err)
				if err != nil {
					err = NewMultiError(err, afterErr)
				} else {
					err = afterErr
				}
			}
		}()
	}

	if a.Before != nil {
		beforeErr := a.Before(context)
		if beforeErr != nil {
			HandleExitCoder(beforeErr)
			err = beforeErr
			return err
		}
	}

	args := context.Args()
	if args.Present() {
		name := args.First()
		c := a.Command(name)
		if c != nil {
			return c.Run(context)
		}
	}

	err = HandleAction(a.Action, context)

	HandleExitCoder(err)
	return err
}

func (a *App) Command(name string) *Command {
	for _, c := range a.Commands {
		if c.HasName(name) {
			return &c
		}
	}

	return nil
}

func (a *App) Categories() CommandCategories {
	return a.categories
}

func (a *App) VisibleCategories() []*CommandCategory {
	ret := []*CommandCategory{}
	for _, category := range a.categories {
		if visible := func() *CommandCategory {
			for _, command := range category.Commands {
				if !command.Hidden {
					return category
				}
			}
			return nil
		}(); visible != nil {
			ret = append(ret, visible)
		}
	}
	return ret
}

func (a *App) VisibleCommands() []Command {
	ret := []Command{}
	for _, command := range a.Commands {
		if !command.Hidden {
			ret = append(ret, command)
		}
	}
	return ret
}

func (a *App) VisibleFlags() []Flag {
	return visibleFlags(a.Flags)
}

func (a *App) hasFlag(flag Flag) bool {
	for _, f := range a.Flags {
		if flag == f {
			return true
		}
	}

	return false
}

func (a *App) errWriter() io.Writer {

	if a.ErrWriter == nil {
		return ErrWriter
	}

	return a.ErrWriter
}

func (a *App) appendFlag(flag Flag) {
	if !a.hasFlag(flag) {
		a.Flags = append(a.Flags, flag)
	}
}

type Author struct {
	Name  string 
	Email string 
}

func (a Author) String() string {
	e := ""
	if a.Email != "" {
		e = " <" + a.Email + ">"
	}

	return fmt.Sprintf("%v%v", a.Name, e)
}

func HandleAction(action interface{}, context *Context) (err error) {
	if a, ok := action.(ActionFunc); ok {
		return a(context)
	} else if a, ok := action.(func(*Context) error); ok {
		return a(context)
	} else if a, ok := action.(func(*Context)); ok { 
		a(context)
		return nil
	} else {
		return errInvalidActionType
	}
}
