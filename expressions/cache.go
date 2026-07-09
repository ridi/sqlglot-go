package expressions

// Cache/Uncache back Spark's `CACHE [LAZY] TABLE x [OPTIONS(k = v)] [AS <query>]` /
// `UNCACHE TABLE [IF EXISTS] x` (core.py:1583-1594). Both are plain Expression
// subclasses, so each is a one-line constructor per the node-model convention
// (AGENTS.md:34-43).
func Cache(args Args) Expression   { return newNode(KindCache, args) }
func Uncache(args Args) Expression { return newNode(KindUncache, args) }
