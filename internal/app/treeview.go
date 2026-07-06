package app

import (
	"fmt"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type overviewTreeSelection struct {
	view     *tview.TreeView
	onSelect func(reference any)
}

type overviewTreeNode struct {
	Label    string
	Value    any
	Children []overviewTreeNode
}

func overviewTreeView(title string, root overviewTreeNode, footer string) (tview.Primitive, *tview.TreeView) {
	view := tview.NewTreeView()
	view.SetBorder(true)
	view.SetTitle(title)
	view.SetBackgroundColor(tcell.ColorBlack)
	view.SetBorderColor(tcell.ColorCadetBlue)

	rootNode := buildOverviewTreeNode(root)
	rootNode.SetExpanded(true)
	expandOverviewTree(rootNode)
	view.SetRoot(rootNode)
	view.SetCurrentNode(rootNode)

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.AddItem(view, 0, 1, true)
	container.AddItem(overviewFooter(footer), 1, 0, false)
	return container, view
}

func buildOverviewTreeNode(node overviewTreeNode) *tview.TreeNode {
	treeNode := tview.NewTreeNode(node.Label)
	treeNode.SetReference(node.Value)
	for _, child := range node.Children {
		treeNode.AddChild(buildOverviewTreeNode(child))
	}
	return treeNode
}

func expandOverviewTree(node *tview.TreeNode) {
	if node == nil {
		return
	}
	node.SetExpanded(true)
	for _, child := range node.GetChildren() {
		expandOverviewTree(child)
	}
}

func templateTreeNode(node client.TemplateTreeNode) overviewTreeNode {
	label := node.Name
	if node.Status != "" {
		label = fmt.Sprintf("%s [%s]", node.Name, node.Status)
	}
	children := make([]overviewTreeNode, 0, len(node.Children))
	for _, child := range node.Children {
		children = append(children, templateTreeNode(child))
	}
	return overviewTreeNode{Label: label, Value: node, Children: children}
}

func resourceTreeNode(node client.ResourceTreeNode) overviewTreeNode {
	label := node.Name
	if node.State != "" || node.Status != "" {
		label = fmt.Sprintf("%s [%s/%s]", node.Name, blankDash(node.State), blankDash(node.Status))
	}
	if node.TemplateName != "" {
		label = fmt.Sprintf("%s (%s)", label, node.TemplateName)
	}
	children := make([]overviewTreeNode, 0, len(node.Children))
	for _, child := range node.Children {
		children = append(children, resourceTreeNode(child))
	}
	return overviewTreeNode{Label: label, Value: node, Children: children}
}
