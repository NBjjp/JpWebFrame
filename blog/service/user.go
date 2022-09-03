package service

import (
	"fmt"
	"github.com/NBjjp/JpWebFrame/orm"
	_ "github.com/go-sql-driver/mysql"
	"net/url"
)

//type User struct {
//	Id       int64  `jporm:"id,auto_increment"`
//	Username string `jporm:"user_name"`
//	Password string `jporm:"password"`
//	Age      int    `jporm:"age"`
//}

type User struct {
	Id       int64
	UserName string
	Password string
	Age      int
}

//func SaveUser() {
//	dataSourceName := fmt.Sprintf("root:123456@tcp(localhost:3306)/test?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
//	db := orm.Open("mysql", dataSourceName)
//	db.Prefix = ""
//	user := &User{
//		//Id:       1000,
//		UserName: "jjp",
//		Password: "123456",
//		Age:      30,
//	}
//	id, _, err := db.NewSession().Insert(user)
//	if err != nil {
//		panic(err)
//	}
//	fmt.Println(id)
//	db.Close()
//}

//批量插入
//func SaveUserBatch() {
//	dataSourceName := fmt.Sprintf("root:123456@tcp(localhost:3306)/test?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
//	db := orm.Open("mysql", dataSourceName)
//	db.Prefix = ""
//	user := &User{
//		//Id:       1000,
//		UserName: "jjp",
//		Password: "123456",
//		Age:      30,
//	}
//	user1 := &User{
//		//Id:       1000,
//		UserName: "jjp1",
//		Password: "1234561",
//		Age:      301,
//	}
//	var users []any
//	users = append(users, user, user1)
//	id, _, err := db.NewSession(user).InsertBatch(users)
//	if err != nil {
//		panic(err)
//	}
//	fmt.Println(id)
//	db.Close()
//}

//func UpdateUser() {
//	dataSourceName := fmt.Sprintf("root:123456@tcp(localhost:3306)/test?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
//	db := orm.Open("mysql", dataSourceName)
//	db.Prefix = ""
//	//user := &User{
//	//	//Id:       1000,
//	//	UserName: "wdf",
//	//	Password: "123456",
//	//	Age:      300,
//	//}
//	//id, _, err := db.NewSession().Where("id", 1018).Where("age", 301).UpDate(user)
//	id, _, err := db.NewSession(user).Where("id", 1018).UpDateParam("age", 100).UpDate()
//
//	if err != nil {
//		panic(err)
//	}
//	fmt.Println(id)
//	db.Close()
//}

func SelectOne() {
	dataSourceName := fmt.Sprintf("root:123456@tcp(localhost:3306)/test?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	db.Prefix = ""
	user := &User{}
	err := db.NewSession(user).Where("age", 301).SelectOne(user)
	if err != nil {
		panic(err)
	}
	fmt.Println(user)
	db.Close()
}
func Select() {
	dataSourceName := fmt.Sprintf("root:123456@tcp(localhost:3306)/test?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	db.Prefix = ""
	user := &User{}
	users, err := db.NewSession(user).OrderDesc("id").Select(user)
	if err != nil {
		panic(err)
	}
	for _, v := range users {
		u := v.(*User)
		fmt.Println(u)
	}
	db.Close()
}
func Count() {
	dataSourceName := fmt.Sprintf("root:123456@tcp(localhost:3306)/test?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	db.Prefix = ""
	user := &User{}
	count, err := db.NewSession(user).Count()
	if err != nil {
		panic(err)
	}
	fmt.Println(count)
	db.Close()
}
