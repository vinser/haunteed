package quit

type Model struct {
}

func New() Model {
	return Model{}
}

func (m Model) View() string {
	return "\nIt's a pity you gave up :(\nBye!\n"
}
