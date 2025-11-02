package main

import "BuildRun/server"

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

func main() {
	svr := server.NewServer("0.0.0.0", 9000)
	svr.Start()
}
