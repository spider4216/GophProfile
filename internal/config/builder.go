package config

type Builder[T any] struct {
	target *T
	steps  []func(*T) error
}

func NewBuilder[T any](target *T) *Builder[T] {
	return &Builder[T]{
		target: target,
	}
}

func (b *Builder[T]) Step(fn func(*T) error) *Builder[T] {
	b.steps = append(b.steps, fn)

	return b
}

func (b *Builder[T]) Build() (*T, error) {
	for _, step := range b.steps {
		if err := step(b.target); err != nil {
			return nil, err
		}
	}

	return b.target, nil
}
