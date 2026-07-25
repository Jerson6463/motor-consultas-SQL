package planner

import (
	"strconv"
	"strings"

	"motor-consultas-sql/internal/parser"
)

// Explain describe un plan logico como un arbol con sangria, de la raiz hacia
// las hojas:
//
//	Project(nombre, salario)
//	└── Filter(edad >= 18)
//	    └── Scan(empleados)
//
// Un join, que tiene dos entradas, se dibuja con ambas ramas:
//
//	HashJoin(a.id = b.id)
//	├── Scan(a)
//	└── Scan(b)
func Explain(node Node) string {
	var builder strings.Builder
	builder.WriteString(describe(node))
	builder.WriteString("\n")
	writeChildren(&builder, node, "")
	return builder.String()
}

func writeChildren(builder *strings.Builder, node Node, prefix string) {
	children := inputs(node)
	for index, child := range children {
		last := index == len(children)-1

		builder.WriteString(prefix)
		if last {
			builder.WriteString("└── ")
		} else {
			builder.WriteString("├── ")
		}
		builder.WriteString(describe(child))
		builder.WriteString("\n")

		// Las ramas que aun tienen hermanos por debajo prolongan la linea.
		if last {
			writeChildren(builder, child, prefix+"    ")
		} else {
			writeChildren(builder, child, prefix+"│   ")
		}
	}
}

// describe da la etiqueta de un nodo.
func describe(node Node) string {
	switch node := node.(type) {
	case *Scan:
		if node.Alias != "" {
			return "Scan(" + node.Alias + ")"
		}
		return "Scan(" + node.Table.Name + ")"
	case *Join:
		return node.Strategy.String() + "Join(" + parser.Format(node.Condition) + ")"
	case *Filter:
		return "Filter(" + parser.Format(node.Condition) + ")"
	case *Aggregate:
		return "Aggregate(" + describeAggregate(node) + ")"
	case *Project:
		return "Project(" + describeProject(node) + ")"
	case *Sort:
		return "Sort(" + describeSort(node) + ")"
	case *Distinct:
		return "Distinct"
	case *Limit:
		return "Limit(" + describeLimit(node) + ")"
	default:
		return "?"
	}
}

// inputs devuelve los hijos de un nodo, en orden.
func inputs(node Node) []Node {
	switch node := node.(type) {
	case *Join:
		return []Node{node.Left, node.Right}
	case *Filter:
		return []Node{node.Input}
	case *Aggregate:
		return []Node{node.Input}
	case *Project:
		return []Node{node.Input}
	case *Sort:
		return []Node{node.Input}
	case *Distinct:
		return []Node{node.Input}
	case *Limit:
		return []Node{node.Input}
	default:
		return nil
	}
}

func describeProject(node *Project) string {
	names := make([]string, len(node.Items))
	for index, item := range node.Items {
		if item.Star {
			names[index] = "*"
			continue
		}
		names[index] = item.Name
	}
	return strings.Join(names, ", ")
}

func describeAggregate(node *Aggregate) string {
	parts := []string{}
	if len(node.GroupBy) > 0 {
		keys := make([]string, len(node.GroupBy))
		for index, expression := range node.GroupBy {
			keys[index] = parser.Format(expression)
		}
		parts = append(parts, "por "+strings.Join(keys, ", "))
	}
	for _, call := range node.Calls {
		parts = append(parts, parser.Format(call))
	}
	return strings.Join(parts, ", ")
}

func describeSort(node *Sort) string {
	terms := make([]string, len(node.Terms))
	for index, term := range node.Terms {
		terms[index] = parser.Format(term.Expression)
		if term.Descending {
			terms[index] += " DESC"
		}
	}
	return strings.Join(terms, ", ")
}

func describeLimit(node *Limit) string {
	parts := []string{}
	if node.Max >= 0 {
		parts = append(parts, "max "+strconv.Itoa(node.Max))
	}
	if node.Offset > 0 {
		parts = append(parts, "offset "+strconv.Itoa(node.Offset))
	}
	return strings.Join(parts, ", ")
}
