package examples

import (
	"context"
	"regexp"
	"testing"

	"github.com/RevenueMonster/sqlike/sql"
	"github.com/RevenueMonster/sqlike/sql/expr"
	"github.com/RevenueMonster/sqlike/sqlike"
	"github.com/stretchr/testify/require"
)

// QueryExamples :
func QueryExamples(ctx context.Context, t *testing.T, db *sqlike.Database) {

	stmt := expr.Union(
		sql.Select().
			From("sqlike", "GeneratedStruct").
			Where(
				expr.Equal("State", ""),
				expr.GreaterThan("Date.CreatedAt", "2006-01-02 16:00:00"),
				expr.GreaterOrEqual("No", 0),
				expr.Or(
					expr.Equal("State", ""),
					expr.NotEqual("State", ""),
					expr.GreaterOrEqual("State", ""),
				),
			).
			OrderBy(
				expr.Desc("NestedID"),
				expr.Asc("Amount"),
			).
			Limit(1).
			Offset(1),
		sql.Select().
			From("sqlike", "GeneratedStruct").
			Limit(1),
	)

	{
		// table := db.Table("GeneratedStruct")
		// err = table.Truncate()
		// require.NoError(t, err)

		result, err := db.QueryStmt(ctx, stmt)
		require.NoError(t, err)
		defer result.Close()

		gss := make([]*generatedStruct, 0)
		for result.Next() {
			gs := new(generatedStruct)
			if err := result.Decode(gs); err != nil {
				panic(err)
			}
			gss = append(gss, gs)
		}

		// TODO: add test
	}

	{
		if err := db.RunInTransaction(
			ctx,
			func(sess sqlike.SessionContext) error {
				if _, err := sess.Exec("USE `sqlike`;"); err != nil {
					return err
				}

				var version string
				if err := sess.QueryRow(`SELECT VERSION();`).Scan(&version); err != nil {
					return err
				}
				require.Regexp(t, regexp.MustCompile(`\d+\.\d+\d+`), version)

				rows, err := sess.Query("SELECT COUNT(*) FROM `GeneratedStruct`;")
				if err != nil {
					return err
				}

				var count uint
				for rows.Next() {
					if err := rows.Scan(&count); err != nil {
						return err
					}
				}
				require.NotEmpty(t, count)

				if err := rows.Close(); err != nil {
					return err
				}

				result, err := sess.QueryStmt(stmt)
				if err != nil {
					return err
				}
				defer result.Close()

				return nil
			}); err != nil {
			panic(err)
		}
	}
}
