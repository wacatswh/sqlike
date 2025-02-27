package mysql

import (
	"github.com/RevenueMonster/sqlike/sqlike/options"
	"github.com/RevenueMonster/sqlike/util"
)

// Connect :
func (ms MySQL) Connect(opt *options.ConnectOptions) (connStr string) {
	if opt.RawConnStr() != "" {
		connStr = opt.RawConnStr()
		return
	}

	if opt.Username == "" {
		panic("missing username for db connection")
	}

	blr := util.AcquireString()
	defer util.ReleaseString(blr)
	blr.WriteString(opt.Username)
	blr.WriteByte(':')
	blr.WriteString(opt.Password)
	blr.WriteByte('@')
	if opt.Socket != "" {
		blr.WriteString(opt.Socket)
	} else {
		if opt.Protocol != "" {
			blr.WriteString(opt.Protocol)
		} else {
			blr.WriteString("tcp")
		}
		blr.WriteByte('(')
		blr.WriteString(opt.Host)
		if opt.Port != "" {
			blr.WriteByte(':')
			blr.WriteString(opt.Port)
		}
		blr.WriteByte(')')
	}
	blr.WriteByte('/')
	blr.WriteByte('?')
	blr.WriteString("parseTime=true")
	if opt.Charset == "" {
		blr.WriteString("&charset=utf8mb4")
		blr.WriteString("&collation=utf8mb4_unicode_ci")
	} else {
		blr.WriteString("&charset=" + string(opt.Charset))
		if opt.Collate != "" {
			blr.WriteString("&collation=" + string(opt.Collate))
		}
	}
	connStr = blr.String()
	return
}
