package orm

import (
	"database/sql"
	"errors"
	"fmt"
	jplog "github.com/NBjjp/JpWebFrame/log"
	"reflect"
	"strings"
	"time"
)

//orm的本质就是拼接sql，让开发者更加方便的使用

type JpDb struct {
	db     *sql.DB
	logger *jplog.Logger
	Prefix string
}
type JpSession struct {
	db        *JpDb
	tx        *sql.Tx
	beginTx   bool
	tableName string
	//字段
	fieleName []string
	//占位符
	placeHolder []string
	values      []any
	updateParam strings.Builder
	whereParam  strings.Builder
	whereValues []any
}

func Open(driverName string, source string) *JpDb {
	db, err := sql.Open(driverName, source)
	if err != nil {
		panic(err)
	}
	jpDb := &JpDb{
		db:     db,
		logger: jplog.Default(),
	}
	//最大空闲连接数，默认不配置，是2个最大空闲连接
	db.SetMaxIdleConns(5)
	//最大连接数，默认不配置，是不限制最大连接数
	db.SetMaxOpenConns(100)
	// 连接最大存活时间
	db.SetConnMaxLifetime(time.Minute * 3)
	//空闲连接最大存活时间
	db.SetConnMaxIdleTime(time.Minute * 1)
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	return jpDb
}

func (db *JpDb) Close() error {
	return db.db.Close()
}

//SetMaxIdleConns 最大空闲连接数，默认不配置，是2个最大空闲连接
func (db *JpDb) SetMaxIdleConns(n int) {
	db.db.SetMaxIdleConns(n)
}
func (db *JpDb) NewSession(data any) *JpSession {
	J := &JpSession{
		db: db,
	}
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Pointer {
		panic(errors.New("data must be pointer"))
	}
	tVar := t.Elem()
	if J.tableName == "" {
		J.tableName = J.db.Prefix + strings.ToLower(tVar.Name())
	}
	return J
}
func (s *JpSession) Table(name string) *JpSession {
	s.tableName = name
	return s
}
func (s *JpSession) Insert(data any) (int64, int64, error) {
	s.fieldNames(data)
	//每一个操作都是独立的  互不影响的   每一次操作都是在会话中完成
	//insert into table (xxx,xxx) values (?,?)
	query := fmt.Sprintf("insert into %s (%s) values (%s)", s.tableName, strings.Join(s.fieleName, ","), strings.Join(s.placeHolder, ","))
	fmt.Println(query)
	s.db.logger.Info(query)
	var stmt *sql.Stmt
	var err error
	if s.beginTx {
		stmt, err = s.tx.Prepare(query)
	} else {
		stmt, err = s.db.db.Prepare(query)
	}
	if err != nil {
		return -1, -1, err
	}
	r, err := stmt.Exec(s.values...)
	if err != nil {
		return -1, -1, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		return -1, -1, err
	}
	return id, affected, nil
}

func (s *JpSession) fieldNames(data any) {
	//反射
	t := reflect.TypeOf(data)
	v := reflect.ValueOf(data)
	if t.Kind() != reflect.Pointer {
		panic(errors.New("data must be pointer"))
	}
	tVar := t.Elem()
	vVar := v.Elem()
	if s.tableName == "" {
		s.tableName = s.db.Prefix + strings.ToLower(tVar.Name())
	}
	for i := 0; i < tVar.NumField(); i++ {
		//字段名称
		fieldName := tVar.Field(i).Name
		tag := tVar.Field(i).Tag
		sqlTag := tag.Get("jporm")
		if sqlTag == "" {
			sqlTag = strings.ToLower(Name(fieldName))

		} else {
			if strings.Contains(sqlTag, "auto_increment") {
				//自增长的主键id
				continue
			}
			if strings.Contains(sqlTag, ",") {
				sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
			}
		}
		id := vVar.Field(i).Interface()
		if strings.ToLower(sqlTag) == "id" && isAutoId(id) {
			//对id做个判断 如果其值小于等于0 数据库可能是自增 跳过此字段
			continue
		}
		s.fieleName = append(s.fieleName, sqlTag)
		s.placeHolder = append(s.placeHolder, "?")
		s.values = append(s.values, vVar.Field(i).Interface())
	}
}

func (s *JpSession) InsertBatch(data []any) (int64, int64, error) {
	//insert into table (xxx,xxx) values (?,?)(?,?)(?,?)
	if len(data) == 0 {
		return -1, -1, errors.New("no data insert")
	}
	s.fieldNames(data[0])
	//每一个操作都是独立的  互不影响的   每一次操作都是在会话中完成
	//insert into table (xxx,xxx) values (?,?)
	query := fmt.Sprintf("insert into %s (%s) values ", s.tableName, strings.Join(s.fieleName, ","))
	var sb strings.Builder
	sb.WriteString(query)
	for index, _ := range data {
		sb.WriteString("(")
		sb.WriteString(strings.Join(s.placeHolder, ","))
		sb.WriteString(")")
		if index < len(data)-1 {
			sb.WriteString(",")
		}
	}
	s.batchValues(data)
	s.db.logger.Info(sb.String())
	var stmt *sql.Stmt
	var err error
	if s.beginTx {
		stmt, err = s.tx.Prepare(sb.String())
	} else {
		stmt, err = s.db.db.Prepare(sb.String())
	}
	if err != nil {
		return -1, -1, err
	}
	r, err := stmt.Exec(s.values...)
	if err != nil {
		return -1, -1, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		return -1, -1, err
	}
	return id, affected, nil
}

func (s *JpSession) batchValues(data []any) {
	s.values = make([]any, 0)
	for _, val := range data {
		//反射
		t := reflect.TypeOf(val)
		v := reflect.ValueOf(val)
		if t.Kind() != reflect.Pointer {
			panic(errors.New("data must be pointer"))
		}
		tVar := t.Elem()
		vVar := v.Elem()
		for i := 0; i < tVar.NumField(); i++ {
			//字段名称
			fieldName := tVar.Field(i).Name
			tag := tVar.Field(i).Tag
			sqlTag := tag.Get("jporm")
			if sqlTag == "" {
				sqlTag = strings.ToLower(Name(fieldName))

			} else {
				if strings.Contains(sqlTag, "auto_increment") {
					//自增长的主键id
					continue
				}
				if strings.Contains(sqlTag, ",") {
					sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
				}
			}
			id := vVar.Field(i).Interface()
			if strings.ToLower(sqlTag) == "id" && isAutoId(id) {
				//对id做个判断 如果其值小于等于0 数据库可能是自增 跳过此字段
				continue
			}
			s.values = append(s.values, vVar.Field(i).Interface())
		}
	}
}

func (s *JpSession) UpDateParam(field string, values any) *JpSession {
	if s.updateParam.String() != "" {
		s.updateParam.WriteString(",")
	}
	s.updateParam.WriteString(field)
	s.updateParam.WriteString(" = ? ")
	s.values = append(s.values, values)
	return s
}
func (s *JpSession) UpDateMap(data map[string]any) *JpSession {
	for k, v := range data {
		if s.updateParam.String() != "" {
			s.updateParam.WriteString(",")
		}
		s.updateParam.WriteString(k)
		s.updateParam.WriteString(" = ? ")
		s.values = append(s.values, v)
	}
	return s
}
func (s *JpSession) UpDate(data ...any) (int64, int64, error) {
	//updata ("age",1) or updata(user)
	if len(data) > 2 {
		return -1, -1, errors.New("param not valid")
	}
	if len(data) == 0 {
		sqlString := fmt.Sprintf("update %s set %s", s.tableName, s.updateParam.String())
		var sb strings.Builder
		sb.WriteString(sqlString)
		sb.WriteString(s.whereParam.String())
		s.db.logger.Info(sb.String())
		var stmt *sql.Stmt
		var err error
		if s.beginTx {
			stmt, err = s.tx.Prepare(sb.String())
		} else {
			stmt, err = s.db.db.Prepare(sb.String())
		}
		if err != nil {
			return -1, -1, err
		}
		s.values = append(s.values, s.whereValues...)
		r, err := stmt.Exec(s.values...)
		if err != nil {
			return -1, -1, err
		}
		id, err := r.LastInsertId()
		if err != nil {
			return -1, -1, err
		}
		affected, err := r.RowsAffected()
		if err != nil {
			return -1, -1, err
		}
		return id, affected, nil
	}
	single := true
	if len(data) == 2 {
		//updata ("age",1)
		single = false
	}
	//updata table set age=?,name=?,where id=?
	if !single {
		if s.updateParam.String() != "" {
			s.updateParam.WriteString(",")
		}
		s.updateParam.WriteString(data[0].(string))
		s.updateParam.WriteString(" = ? ")
		s.values = append(s.values, data[1])
	} else {
		//updata(user)
		updatedata := data[0]
		//反射
		t := reflect.TypeOf(updatedata)
		v := reflect.ValueOf(updatedata)
		if t.Kind() != reflect.Pointer {
			panic(errors.New("updatedata must be pointer"))
		}
		tVar := t.Elem()
		vVar := v.Elem()
		if s.tableName == "" {
			s.tableName = s.db.Prefix + strings.ToLower(tVar.Name())
		}
		for i := 0; i < tVar.NumField(); i++ {
			//字段名称
			fieldName := tVar.Field(i).Name
			tag := tVar.Field(i).Tag
			sqlTag := tag.Get("jporm")
			if sqlTag == "" {
				sqlTag = strings.ToLower(Name(fieldName))
			} else {
				if strings.Contains(sqlTag, "auto_increment") {
					//自增长的主键id
					continue
				}
				if strings.Contains(sqlTag, ",") {
					sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
				}
			}
			id := vVar.Field(i).Interface()
			if strings.ToLower(sqlTag) == "id" && isAutoId(id) {
				//对id做个判断 如果其值小于等于0 数据库可能是自增 跳过此字段
				continue
			}
			if s.updateParam.String() != "" {
				s.updateParam.WriteString(",")
			}
			s.updateParam.WriteString(sqlTag)
			s.updateParam.WriteString(" = ? ")
			s.values = append(s.values, vVar.Field(i).Interface())
		}
	}
	sqlString := fmt.Sprintf("update %s set %s", s.tableName, s.updateParam.String())
	var sb strings.Builder
	sb.WriteString(sqlString)
	sb.WriteString(s.whereParam.String())
	s.db.logger.Info(sb.String())
	var stmt *sql.Stmt
	var err error
	if s.beginTx {
		stmt, err = s.tx.Prepare(sb.String())
	} else {
		stmt, err = s.db.db.Prepare(sb.String())
	}
	if err != nil {
		return -1, -1, err
	}
	s.values = append(s.values, s.whereValues...)
	r, err := stmt.Exec(s.values...)
	if err != nil {
		return -1, -1, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		return -1, -1, err
	}
	return id, affected, nil
}

//select * from table where id=1000
func (s *JpSession) SelectOne(data any, fields ...string) error {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Pointer {
		return errors.New("data must be point")
	}
	fieldStr := "*"
	if len(fields) > 0 {
		fieldStr = strings.Join(fields, ",")
	}
	sqlString := fmt.Sprintf("select %s from %s", fieldStr, s.tableName)
	var sb strings.Builder
	sb.WriteString(sqlString)
	sb.WriteString(s.whereParam.String())
	s.db.logger.Info(sb.String())
	stmt, err := s.db.db.Prepare(sb.String())
	if err != nil {
		return err
	}
	rows, err := stmt.Query(s.whereValues...)
	if err != nil {
		return err
	}
	//id user_name age
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	values := make([]any, len(columns))
	fieldsScan := make([]any, len(columns))
	for i := range fieldsScan {
		fieldsScan[i] = &values[i]
	}
	if rows.Next() {
		err := rows.Scan(fieldsScan...)
		if err != nil {
			return err
		}
		tVar := t.Elem()
		vVar := reflect.ValueOf(data).Elem()
		for i := 0; i < tVar.NumField(); i++ {
			name := tVar.Field(i).Name
			tag := tVar.Field(i).Tag
			sqlTag := tag.Get("jporm")
			if sqlTag == "" {
				sqlTag = strings.ToLower(Name(name))
			} else {
				if strings.Contains(sqlTag, ",") {
					sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
				}
			}
			for j, colName := range columns {
				if sqlTag == colName {
					target := values[j]
					targetValue := reflect.ValueOf(target)
					fieldType := tVar.Field(i).Type
					result := reflect.ValueOf(targetValue.Interface()).Convert(fieldType)
					vVar.Field(i).Set(result)
				}
			}
		}
	}
	return nil
}

//查询多行
func (s *JpSession) Select(data any, fields ...string) ([]any, error) {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Pointer {
		return nil, errors.New("data must be point")
	}
	fieldStr := "*"
	if len(fields) > 0 {
		fieldStr = strings.Join(fields, ",")
	}
	sqlString := fmt.Sprintf("select %s from %s", fieldStr, s.tableName)
	var sb strings.Builder
	sb.WriteString(sqlString)
	sb.WriteString(s.whereParam.String())
	s.db.logger.Info(sb.String())
	stmt, err := s.db.db.Prepare(sb.String())
	if err != nil {
		return nil, err
	}
	rows, err := stmt.Query(s.whereValues...)
	if err != nil {
		return nil, err
	}
	//id user_name age
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := make([]any, 0)
	for {
		if rows.Next() {
			//由于传进来的是指针地址，如果每次赋值，实际都是一个result   每个值都一样
			//每次查询时 data重新换一个地址
			data := reflect.New(t.Elem()).Interface()
			values := make([]any, len(columns))
			fieldsScan := make([]any, len(columns))
			for i := range fieldsScan {
				fieldsScan[i] = &values[i]
			}
			err := rows.Scan(fieldsScan...)
			if err != nil {
				return nil, err
			}
			tVar := t.Elem()
			vVar := reflect.ValueOf(data).Elem()
			for i := 0; i < tVar.NumField(); i++ {
				name := tVar.Field(i).Name
				tag := tVar.Field(i).Tag
				sqlTag := tag.Get("jporm")
				if sqlTag == "" {
					sqlTag = strings.ToLower(Name(name))
				} else {
					if strings.Contains(sqlTag, ",") {
						sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
					}
				}
				for j, colName := range columns {
					if sqlTag == colName {
						target := values[j]
						targetValue := reflect.ValueOf(target)
						fieldType := tVar.Field(i).Type
						result := reflect.ValueOf(targetValue.Interface()).Convert(fieldType)
						vVar.Field(i).Set(result)
					}
				}
			}
			result = append(result, data)
		} else {
			break
		}
	}
	return result, nil
}

//删除
func (s *JpSession) Delete() (int64, error) {
	//delete from table where id=?
	sqlString := fmt.Sprintf("delete from %s ", s.tableName)
	var sb strings.Builder
	sb.WriteString(sqlString)
	sb.WriteString(s.whereParam.String())
	s.db.logger.Info(sb.String())
	var stmt *sql.Stmt
	var err error
	if s.beginTx {
		stmt, err = s.tx.Prepare(sqlString)
	} else {
		stmt, err = s.db.db.Prepare(sqlString)
	}
	if err != nil {
		return 0, nil
	}
	result, err := stmt.Exec(s.whereValues...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

//聚合函数
func (s *JpSession) Count() (int64, error) {
	return s.Aggregate("count", "*")
}

//聚合函数通用实现
func (s *JpSession) Aggregate(funcName string, field string) (int64, error) {
	var fieldSb strings.Builder
	fieldSb.WriteString(funcName)
	fieldSb.WriteString("(")
	fieldSb.WriteString(field)
	fieldSb.WriteString(")")
	sqlString := fmt.Sprintf("select %s from %s", fieldSb.String(), s.tableName)
	var sb strings.Builder
	sb.WriteString(sqlString)
	sb.WriteString(s.whereParam.String())
	s.db.logger.Info(sb.String())
	stmt, err := s.db.db.Prepare(sb.String())
	if err != nil {
		return 0, err
	}
	row := stmt.QueryRow(s.whereValues...)
	if row.Err() != nil {
		return 0, err
	}
	var result int64
	err = row.Scan(&result)
	if err != nil {
		return 0, err
	}
	return result, nil
}

//原生SQL的支持
func (s *JpSession) Exec(query string, values ...any) (int64, error) {
	var stmt *sql.Stmt
	var err error
	if s.beginTx {
		stmt, err = s.tx.Prepare(query)
	} else {
		stmt, err = s.db.db.Prepare(query)
	}
	if err != nil {
		return 0, err
	}
	r, err := stmt.Exec(values)
	if err != nil {
		return 0, err
	}
	if strings.Contains(strings.ToLower(query), "insert") {
		return r.LastInsertId()
	}
	return r.RowsAffected()
}
func (s *JpSession) QueryRow(sql string, data any, queryValues ...any) error {
	t := reflect.TypeOf(data)
	stmt, err := s.db.db.Prepare(sql)
	if err != nil {
		return err
	}
	rows, err := stmt.Query(queryValues...)
	if err != nil {
		return err
	}
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	values := make([]any, len(columns))
	var fieldsScan = make([]any, len(columns))
	for i := range fieldsScan {
		fieldsScan[i] = &values[i]
	}
	if rows.Next() {
		err = rows.Scan(fieldsScan...)
		if err != nil {
			return err
		}
		v := reflect.ValueOf(data)
		valueOf := reflect.ValueOf(values)
		for i := 0; i < t.Elem().NumField(); i++ {
			name := t.Elem().Field(i).Name
			tag := t.Elem().Field(i).Tag
			sqlTag := tag.Get("msorm")
			if sqlTag == "" {
				sqlTag = strings.ToLower(Name(name))
			} else {
				if strings.Contains(sqlTag, ",") {
					sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
				}
			}
			for j, coName := range columns {
				if sqlTag == coName {
					if v.Elem().Field(i).CanSet() {
						covertValue := s.ConvertType(valueOf, j, v, i)
						v.Elem().Field(i).Set(covertValue)
					}
				}
			}
		}
	}

	return nil

}

func (s *JpSession) Where(field string, value any) *JpSession {
	//id=1
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" = ? ")
	s.whereValues = append(s.whereValues, value)
	return s
}

//其他查询条件
func (s *JpSession) Like(field string, value any) *JpSession {
	//name like %s%
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" like ")
	s.whereParam.WriteString(" ? ")
	s.whereValues = append(s.whereValues, "%"+value.(string)+"%")
	return s
}
func (s *JpSession) LikeRight(field string, value any) *JpSession {
	//name like %s%
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" like ")
	s.whereParam.WriteString(" ? ")
	s.whereValues = append(s.whereValues, value.(string)+"%")
	return s
}
func (s *JpSession) LikeLeft(field string, value any) *JpSession {
	//name like %s%
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" like ")
	s.whereParam.WriteString(" ? ")
	s.whereValues = append(s.whereValues, "%"+value.(string))
	return s
}
func (s *JpSession) Group(field ...string) *JpSession {
	//group by aa,bb
	s.whereParam.WriteString(" group by ")
	s.whereParam.WriteString(strings.Join(field, ","))
	return s
}
func (s *JpSession) OrderDesc(field ...string) *JpSession {
	//order by aa,bb desc
	s.whereParam.WriteString(" order by ")
	s.whereParam.WriteString(strings.Join(field, ","))
	s.whereParam.WriteString(" desc ")
	return s
}
func (s *JpSession) OrderAsc(field ...string) *JpSession {
	s.whereParam.WriteString(" order by ")
	s.whereParam.WriteString(strings.Join(field, ","))
	s.whereParam.WriteString(" asc ")
	return s
}

//order by aa desc,bb asc
//Order Order("aa" "desc","bb" "asc")
func (s *JpSession) Order(field ...string) *JpSession {
	if len(field)%2 != 0 {
		panic("field num not true")
	}
	s.whereParam.WriteString(" order by ")
	for index, v := range field {
		s.whereParam.WriteString(v + " ")
		if index%2 != 0 && index < len(field)-1 {
			s.whereParam.WriteString(" , ")
		}
	}
	return s
}

//事务支持
func (s *JpSession) Begin() error {
	tx, err := s.db.db.Begin()
	if err != nil {
		return err
	}
	s.tx = tx
	s.beginTx = true
	return nil
}

func (s *JpSession) Commit() error {
	err := s.tx.Commit()
	if err != nil {
		return err
	}
	s.beginTx = false
	return nil
}

func (s *JpSession) Rollback() error {
	err := s.tx.Rollback()
	if err != nil {
		return err
	}
	s.beginTx = false
	return nil
}
func (s *JpSession) And() *JpSession {
	s.whereParam.WriteString(" and ")
	return s
}
func (s *JpSession) Or() *JpSession {
	s.whereParam.WriteString(" or ")
	return s
}

func isAutoId(id any) bool {
	t := reflect.ValueOf(id)
	switch t.Kind() {
	case reflect.Int64:
		if id.(int64) <= 0 {
			return true
		}
	case reflect.Int:
		if id.(int) <= 0 {
			return true
		}
	default:
		return false
	}
	return false
}

//加下划线
func Name(name string) string {
	all := name[:]
	var sb strings.Builder
	lastIndex := 0
	for index, value := range all {
		//大写字母
		if value >= 65 && value <= 90 {
			if index == 0 {
				continue
			}
			sb.WriteString(name[lastIndex:index])
			sb.WriteString("_")
			lastIndex = index
		}
	}
	sb.WriteString(name[lastIndex:])
	return sb.String()
}

func (s *JpSession) ConvertType(valueOf reflect.Value, j int, v reflect.Value, i int) reflect.Value {
	eVar := valueOf.Index(j)
	t2 := v.Elem().Field(i).Type()
	of := reflect.ValueOf(eVar.Interface())
	covertValue := of.Convert(t2)
	return covertValue
}
