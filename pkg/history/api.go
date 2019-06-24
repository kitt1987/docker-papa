package history

type File interface {
	Log(key string, content string)
	Search(key string) []string
	Truncate(size int)
	Close()
}
