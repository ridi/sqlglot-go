package trie

type TrieResult int

const (
	Failed TrieResult = iota
	Prefix
	Exists
)

type Node struct {
	Children map[rune]*Node
	Terminal bool
}

func NewTrie(keywords []string) *Node {
	root := &Node{Children: map[rune]*Node{}}
	for _, keyword := range keywords {
		current := root
		for _, char := range keyword {
			if current.Children == nil {
				current.Children = map[rune]*Node{}
			}
			next := current.Children[char]
			if next == nil {
				next = &Node{Children: map[rune]*Node{}}
				current.Children[char] = next
			}
			current = next
		}
		current.Terminal = true
	}
	return root
}

func InTrie(root *Node, key string) (TrieResult, *Node) {
	if key == "" {
		return Failed, root
	}

	current := root
	for _, char := range key {
		if current == nil || current.Children == nil {
			return Failed, current
		}
		next := current.Children[char]
		if next == nil {
			return Failed, current
		}
		current = next
	}

	if current != nil && current.Terminal {
		return Exists, current
	}
	return Prefix, current
}
