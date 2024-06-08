package storage

type Storage interface {
	Save(magnetLink string) error
}
