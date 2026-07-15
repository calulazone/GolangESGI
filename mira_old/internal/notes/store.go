package notes

type NoteStore interface {
	Save(n *Note) error
	List(limit int) ([]*Note, error)
	All() ([]*Note, error)
}
