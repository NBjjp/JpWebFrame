package frame

import (
	"fmt"
	"testing"
)

func TestTreeNode(t *testing.T) {
	root := &treeNode{
		name:     "/",
		children: make([]*treeNode, 0),
	}
	root.Put("/user/get/:id")
	root.Put("/user/create/dd")
	root.Put("/user/create/id")
	root.Put("/order/get/dd")

	node := root.Get("/user/get/1")
	fmt.Println(node)
	node = root.Get("/user/create/dd")
	fmt.Println(node)
	node = root.Get("/user/create")
	fmt.Println(node)
	node = root.Get("/order/get/dd")
	fmt.Println(node)

}
