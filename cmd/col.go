/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

// colCmd represents the col command

func init() {
	var cmd = &cobra.Command{
		Use:   "col",
		Short: "split csv file by column value",
		Run:   func(cmd *cobra.Command, args []string) {},
	}
	var app = colCmd{Command: cmd}
	cmd.Run = app.Run
	cmd.Args = app.validates()
	rootCmd.AddCommand(app.Command)

	app.bind()
}

type colCmd struct {
	*cobra.Command
	selectedCol int
	maxLine     int
	sorted      bool
	includeTile bool
}

func (cmd *colCmd) bind() {
	cmd.Command.Flags().IntVarP(&cmd.selectedCol, "column", "c", 1, "split by given column")
	cmd.Command.Flags().IntVarP(&cmd.maxLine, "line", "l", 1, "max number of lines per file")
	cmd.Command.Flags().BoolVarP(&cmd.sorted, "sort", "s", false, "sorted by given column value count")
	cmd.Command.Flags().BoolVarP(&cmd.includeTile, "tile", "t", false, "include in splited files")
}
func (c *colCmd) validates() cobra.PositionalArgs {
	return cobra.MatchAll(cobra.ExactArgs(1), func(cmd *cobra.Command, args []string) error {
		if c.maxLine < 1 {
			return errors.New("max line must not less than 1")
		}
		if c.selectedCol < 1 {
			return errors.New("selected column must not less than 1")
		}
		f, err := os.Open(args[0])
		if err != nil {
			return err
		}
		defer f.Close()

		r := csv.NewReader(f)
		line, err := r.Read()
		if err != nil {
			return err
		}

		if len(line) < c.selectedCol {
			return errors.New("column must not greater than total column count")
		}

		return nil
	})
}
func (c *colCmd) Run(cmd *cobra.Command, args []string) {
	file := args[0]
	ext := filepath.Ext(file)
	baseName := file[:len(file)-len(ext)]

	realCol := c.selectedCol - 1

	var titles []string
	var groups = make(map[string][][]string, 128)
	var groupKeys []string
	{
		f, _ := os.Open(file)
		r := csv.NewReader(f)
		lines, err := r.ReadAll()
		if err != nil {
			cmd.PrintErrln("解析文件失败")
			return
		}
		if c.includeTile {
			titles = lines[0]
			lines = lines[1:]
		}
		for _, line := range lines {
			group := line[realCol]
			groupKeys = append(groupKeys, group)
			groups[group] = append(groups[group], line)
		}
	}
	var sortedGroup []string
	if c.sorted {
		var groupCounts = make(map[string]int, len(groups))
		for _, k := range groupKeys {
			groupCounts[k] += 1
		}
		pairs := lo.ToPairs(groupCounts)
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].Value < pairs[j].Value
		})
		sortedGroup = lo.Map(pairs, func(v lo.Entry[string, int], index int) string {
			return v.Key
		})
	} else {
		sortedGroup = lo.Uniq(groupKeys)
	}

	var last [][]string
	var lastGroup string
	var count int
	for _, v := range sortedGroup {
		lastGroup = v
		values := groups[v]
		count += len(values)
		last = append(last, values...)
		if count >= c.maxLine {
			f, err := os.Create(fmt.Sprintf("%s_%s%s", baseName, v, ext))
			if err != nil {
				cmd.PrintErrln("创建新文件失败", err)
				return
			}
			w := csv.NewWriter(f)
			if c.includeTile {
				w.Write(titles)
			}
			w.WriteAll(last)
			w.Flush()

			last = last[:0]
			count = 0
		}
	}
	if len(last) > 0 {
		f, err := os.Create(fmt.Sprintf("%s_%s%s", baseName, lastGroup, ext))
		if err != nil {
			cmd.PrintErrln("创建新文件失败", err)
			return
		}
		w := csv.NewWriter(f)
		if c.includeTile {
			w.Write(titles)
		}
		w.WriteAll(last)
		w.Flush()
	}

}
