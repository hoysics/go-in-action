package orm

import (
	"context"
	"database/sql"
	"errors"
	"github.com/hoysics/go-in-action/orm/select/internal/errs"
)

// Selector 用于构造 SELECT 语句
type Selector[T any] struct {
	builder
	table   TableReference
	where   []Predicate
	having  []Predicate
	columns []Selectable
	groupBy []Column
	offset  int
	limit   int
	sess    session
}

func (s *Selector[T]) Select(cols ...Selectable) *Selector[T] {
	s.columns = cols
	return s
}

// From 指定表名，如果是空字符串，那么将会使用默认表名
func (s *Selector[T]) From(tbl TableReference) *Selector[T] {
	s.table = tbl
	return s
}

func (s *Selector[T]) Build() (*Query, error) {
	var err error
	s.model, err = s.r.Get(new(T))
	if err != nil {
		return nil, err
	}
	s.sb.WriteString("SELECT ")
	if err = s.buildColumns(); err != nil {
		return nil, err
	}
	s.sb.WriteString(" FROM ")
	if err = s.buildTable(s.table); err != nil {
		return nil, err
	}
	// 构造 WHERE
	if len(s.where) > 0 {
		// 类似这种可有可无的部分，都要在前面加一个空格
		s.sb.WriteString(" WHERE ")
		if err = s.buildPredicates(s.where); err != nil {
			return nil, err
		}
	}

	if len(s.groupBy) > 0 {
		s.sb.WriteString(" GROUP BY ")
		for i, c := range s.groupBy {
			if i > 0 {
				s.sb.WriteByte(',')
			}
			if err = s.buildColumn(c, false); err != nil {
				return nil, err
			}
		}
	}

	if len(s.having) > 0 {
		s.sb.WriteString(" HAVING ")
		if err = s.buildPredicates(s.having); err != nil {
			return nil, err
		}
	}

	if s.limit > 0 {
		s.sb.WriteString(" LIMIT ?")
		s.addArgs(s.limit)
	}

	if s.offset > 0 {
		s.sb.WriteString(" OFFSET ?")
		s.addArgs(s.offset)
	}

	s.sb.WriteString(";")
	return &Query{
		SQL:  s.sb.String(),
		Args: s.args,
	}, nil
}

func (s *Selector[T]) buildTable(table TableReference) error {
	switch tbl := table.(type) {
	case nil:
		s.builder.quote(s.model.TableName)
	case Table:
		m, err := s.r.Get(tbl.entity)
		if err != nil {
			return err
		}
		if tbl.alias != "" {
			s.quote(m.TableName)
			s.buildAs(tbl.alias)
		} else {
			s.builder.quote(m.TableName)
		}
	case Join:
		s.sb.WriteByte('(')
		err := s.buildTable(tbl.left)
		if err != nil {
			return err
		}
		s.sb.WriteByte(' ')
		s.sb.WriteString(tbl.typ)
		s.sb.WriteByte(' ')
		err = s.buildTable(tbl.right)
		if err != nil {
			return err
		}
		if len(tbl.on) > 0 {
			s.sb.WriteString(" ON ")
			pre := tbl.on[0]
			for i := 1; i < len(tbl.on); i++ {
				pre = pre.And(tbl.on[i])
			}
			err = s.buildExpr(pre)
			if err != nil {
				return err
			}
		}
		if len(tbl.using) > 0 {
			s.sb.WriteString(" USING (")
			for i, col := range tbl.using {
				if i > 0 {
					s.sb.WriteByte(',')
				}
				err := s.buildColumn(Column{name: col}, false)
				if err != nil {
					return err
				}
			}
			s.sb.WriteByte(')')
		}
		s.sb.WriteByte(')')
	case Subquery:
		s.sb.WriteByte('(')
		q, err := tbl.s.Build()
		if err != nil {
			return err
		}
		s.sb.WriteString(q.SQL[:len(q.SQL)-1])
		s.args = append(s.args, q.Args...)
		s.sb.WriteByte(')')
		s.buildAs(tbl.alias)

	default:
		return errors.New("not supported table type")
	}
	return nil

}

func (s *Selector[T]) buildExpr(p Expression) error {
	switch expr := p.(type) {
	case nil:
		return nil
	case Column:
		err := s.buildColumn(expr, false)
		if err != nil {
			return err
		}
	case value:
		s.sb.WriteByte('?')
		s.args = append(s.args, expr.val)
	case *Predicate:
		_, ok := expr.left.(*Predicate)
		if ok {
			s.sb.WriteString(" (")
		}
		err := s.buildExpr(expr.left)
		if err != nil {
			return err
		}
		if ok {
			s.sb.WriteString(") ")
		}
		s.sb.WriteByte(' ')
		s.sb.WriteString(string(expr.op))
		s.sb.WriteByte(' ')
		_, ok = expr.right.(*Predicate)
		if ok {
			s.sb.WriteString(" (")
		}
		err = s.buildExpr(expr.right)
		if err != nil {
			return err
		}
		if ok {
			s.sb.WriteString(") ")
		}
	case Subquery:
		s.sb.WriteByte('(')
		q, err := expr.Build()
		if err != nil {
			return err
		}
		s.sb.WriteString(q.SQL[:len(q.SQL)-1])

		s.sb.WriteByte(')')
	case SubqueryExpr:
		s.sb.WriteString(expr.pred)
		s.sb.WriteByte(' ')
		err := s.buildExpr(expr.s)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Selector[T]) buildJoin(tab Join) error {
	s.sb.WriteByte('(')
	if err := s.buildTable(tab.left); err != nil {
		return err
	}
	s.sb.WriteString(" ")
	s.sb.WriteString(tab.typ)
	s.sb.WriteString(" ")
	if err := s.buildTable(tab.right); err != nil {
		return err
	}
	if len(tab.using) > 0 {
		s.sb.WriteString(" USING (")
		for i, col := range tab.using {
			if i > 0 {
				s.sb.WriteByte(',')
			}
			err := s.buildColumn(Column{name: col}, false)
			if err != nil {
				return err
			}
		}
		s.sb.WriteString(")")
	}
	if len(tab.on) > 0 {
		s.sb.WriteString(" ON ")
		err := s.buildPredicates(tab.on)
		if err != nil {
			return err
		}
	}
	s.sb.WriteByte(')')
	return nil
}

func (s *Selector[T]) buildColumns() error {
	if len(s.columns) == 0 {
		s.sb.WriteByte('*')
		return nil
	}
	for i, c := range s.columns {
		if i > 0 {
			s.sb.WriteByte(',')
		}
		switch val := c.(type) {
		case Column:
			if err := s.buildColumn(val, true); err != nil {
				return err
			}
		case Aggregate:
			if err := s.buildAggregate(val, true); err != nil {
				return err
			}
		default:
			return errs.NewErrUnsupportedSelectable(val)

		}
	}
	return nil

}

func (s *Selector[T]) buildColumn(c Column, useAlias bool) error {
	err := s.builder.buildColumn(c.table, c.name)
	if err != nil {
		return err
	}
	if useAlias {
		s.buildAs(c.alias)
	}
	return nil
}

// Where 用于构造 WHERE 查询条件。如果 ps 长度为 0，那么不会构造 WHERE 部分
func (s *Selector[T]) Where(ps ...Predicate) *Selector[T] {
	s.where = ps
	return s
}

// GroupBy 设置 group by 子句
func (s *Selector[T]) GroupBy(cols ...Column) *Selector[T] {
	s.groupBy = cols
	return s
}

func (s *Selector[T]) Having(ps ...Predicate) *Selector[T] {
	s.having = ps
	return s
}

func (s *Selector[T]) Offset(offset int) *Selector[T] {
	s.offset = offset
	return s
}

func (s *Selector[T]) Limit(limit int) *Selector[T] {
	s.limit = limit
	return s
}

func (s *Selector[T]) AsSubquery(alias string) Subquery {
	var t T
	tbl := TableOf(&t)
	return Subquery{
		table:   tbl,
		s:       s,
		alias:   alias,
		columns: s.columns,
	}
}

func (s *Selector[T]) Get(ctx context.Context) (*T, error) {
	res := get[T](ctx, s.core, s.sess, &QueryContext{
		Builder: s,
		Type:    "SELECT",
	})
	if res.Result != nil {
		return res.Result.(*T), res.Err
	}
	return nil, res.Err
}

func (s *Selector[T]) GetMulti(ctx context.Context) ([]*T, error) {
	var db sql.DB
	q, err := s.Build()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, q.SQL, q.Args...)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		// 在这里构造 []*T
	}
	panic("implement me")
}

func NewSelector[T any](sess session) *Selector[T] {
	c := sess.getCore()
	return &Selector[T]{
		sess: sess,
		builder: builder{
			core:    c,
			dialect: c.dialect,
			quoter:  c.dialect.quoter(),
		},
	}
}

type Selectable interface {
	selectedAlias() string
	fieldName() string
	target() TableReference
}
