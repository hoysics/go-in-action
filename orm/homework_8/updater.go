package orm

import (
	"context"
	"github.com/hoysics/go-in-action/orm/homework_8/internal/errs"
	"reflect"
)

type Updater[T any] struct {
	builder
	db      *DB
	assigns []Assignable
	val     *T
	where   []Predicate
}

func NewUpdater[T any](db *DB) *Updater[T] {
	return &Updater[T]{
		db: db,
	}
}

func (u *Updater[T]) Update(t *T) *Updater[T] {
	u.val = t
	return u
}

func (u *Updater[T]) Set(assigns ...Assignable) *Updater[T] {
	u.assigns = assigns
	return u
}

func (u *Updater[T]) Build() (*Query, error) {
	if len(u.assigns) == 0 {
		return nil, errs.ErrNoUpdatedColumns
	}
	var t T
	m, err := u.db.r.Get(&t)
	if err != nil {
		return nil, err
	}
	u.model = m

	u.sb.WriteString("UPDATE ")
	u.quote(m.TableName)
	u.sb.WriteString(" SET ")
	val := u.db.valCreator(u.val, u.model)
	for i, assign := range u.assigns {
		switch a := assign.(type) {
		case Assignment:
			if IsZero(a.val) {
				continue
			}
			if err := u.buildColumn(a.column); err != nil {
				return nil, err
			}
			u.sb.WriteString(`=`)
			if err := u.buildExpression(a.val); err != nil {
				return nil, err
			}
		case Column:
			f, err := val.Field(a.name)
			if IsZero(f) {
				continue
			}
			if err != nil {
				return nil, err
			}
			if err = u.buildColumn(a.name); err != nil {
				return nil, err
			}
			u.sb.WriteString(`=?`)
			u.addArgs(f)
		}
		if i < len(u.assigns)-1 {
			u.sb.WriteString(`,`)
		}
	}

	if len(u.where) > 0 {
		// 类似这种可有可无的部分，都要在前面加一个空格
		u.sb.WriteString(" WHERE ")
		// WHERE 是不允许用别名的
		if err = u.buildPredicates(u.where); err != nil {
			return nil, err
		}
	}
	u.sb.WriteString(";")
	return &Query{
		SQL:  u.sb.String(),
		Args: u.args,
	}, nil
}

func (u *Updater[T]) Where(ps ...Predicate) *Updater[T] {
	u.where = ps
	return u
}

func (u *Updater[T]) Exec(ctx context.Context) Result {
	q, err := u.Build()
	if err != nil {
		return Result{err: err}
	}
	res, err := u.db.db.ExecContext(ctx, q.SQL, q.Args...)
	return Result{err: err, res: res}

}

// AssignNotZeroColumns 更新非零值
func AssignNotZeroColumns(entity interface{}) []Assignable {
	panic("implement me")
}

func IsZero(t interface{}) bool {
	val := reflect.ValueOf(t).Elem()
	return val.IsZero()
}
