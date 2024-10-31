package evolviconf

import (
	"context"
	"log/slog"
	"sort"
)

type Position struct {
	Field  string
	Line   int
	Column int
	Value  string
}

type Warnings []Warning

func (w Warnings) Sort() Warnings {
	sort.Slice(w, func(i, j int) bool {
		return w[i].Line < w[j].Line
	})
	return w
}

func (w Warnings) Log(ctx context.Context, logger slog.Logger) {
	for _, ww := range w {
		ww.Log(ctx, logger)
	}
}

type Warning struct {
	Position
	Message string
}

func (w Warning) Log(ctx context.Context, logger slog.Logger) {
	var args []any

	if w.Line != 0 {
		args = append(args, slog.Int("line", w.Line))
	}
	if w.Column != 0 {
		args = append(args, slog.Int("column", w.Column))
	}
	if w.Field != "" {
		args = append(args, slog.String("field", w.Field))
	}
	if w.Value != "" {
		args = append(args, slog.String("value", w.Value))
	}

	logger.WarnContext(ctx, w.Message, args...)
}
