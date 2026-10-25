package main

// ttySize has no answer on Windows yet: the stdlib has no console size
// call, so COLUMNS and LINES, or the default size, stand in.
// ponytail: GetConsoleScreenBufferInfo through syscall once Windows matters.
func ttySize() (int, int, bool) { return 0, 0, false }
