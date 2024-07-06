package lib

const (
	ActionAdd    Action = "add"
	ActionRemove Action = "remove"
	ActionOutput Action = "output"

	IPv4 IPType = "ipv4"
	IPv6 IPType = "ipv6"

	CaseRemovePrefix CaseRemove = 0
	CaseRemoveEntry  CaseRemove = 1
)

var ActionsRegistry = map[Action]bool{
	ActionAdd:    true,
	ActionRemove: true,
	ActionOutput: true,
}

type Action string

type IPType string

type CaseRemove int

type Typer interface {
	GetType() string
}

type Actioner interface {
	GetAction() Action
}

type Descriptioner interface {
	GetDescription() string
}

type InputConverter interface {
	Typer
	Actioner
	Descriptioner
	Input(Container) (Container, error)
}

type OutputConverter interface {
	Typer
	Actioner
	Descriptioner
	Output(Container) error
}

type IgnoreIPOption func() IPType

func IgnoreIPv4() IPType {
	return IPv4
}

func IgnoreIPv6() IPType {
	return IPv6
}
