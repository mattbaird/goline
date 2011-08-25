// Copyright 2011, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
 *  Filename:    goline.go
 *  Package:     goline
 *  Author:      Bryan Matsuo <bmatsuo@soe.ucsc.edu>
 *  Created:     Sat Aug 13 02:28:54 PDT 2011
 *  Description: 
 */

//  Package goline is a command line interfacing (prompting) library inspired
//  by Ruby's HighLine.
//
//  Differences for HighLine users:
//
//      - To be more Go-ish, where HighLine uses the term "strip", the package
//        uses "trim".
//  
//      - Instead of an Agree(question,...) function, the package provides a
//        function `Confirm(question, yesorno) bool`. This is because the
//        author things the term "agree" implies the desire of a positive
//        response to the question ("yes"). The idea is to set up Confirm with
//        positive language and believed truth value of that statement.
//              if cont := false; !Confirm("Continue anyway? ", cont, nil) {
//                  os.Exit(1)
//              }
//              // Continue.
//              // ...
//        But Confirm is flexible enough to be used in other manners.
package goline

import (
    "reflect"
    "strings"
    "unicode"
    "bufio"
    "utf8"
    "fmt"
    "os"
)

//  Returns the index i of the longest terminal substring s[i:] such that f
//  returns true for all runes in s[i:]. Returns -1 if there is no such i.
func stringSuffixIndexFunc(s string, f func(c int) bool) (i int) {
    var hasSuffix bool
    i = strings.LastIndexFunc(s, func(c int) (done bool) {
        if done = !f(c); !hasSuffix {
            hasSuffix = !done
        }
        return
    })
    if i++; !hasSuffix {
        i = -1
    }
    return
}

//  Return the suffix string corresponding to the same call to
//  stringSuffixIndexFunc.
func stringSuffixFunc(s string, f func(c int) bool) (suff string) {
    if i := stringSuffixIndexFunc(s, f); i >= 0 {
        suff = s[i:]
    }
    return
}

func Say(msg string) (int, os.Error) {
    if c, _ := utf8.DecodeLastRuneInString(msg); unicode.IsSpace(c) {
        return fmt.Print(msg)
    }
    return fmt.Println(msg)
}

func SayTrimmed(msg string) (int, os.Error) {
    return Say(strings.TrimRightFunc(msg, unicode.IsSpace))
}

type Stringer interface {
    String() string
}

var (
    zeroStringer Stringer
    typeStringer = reflect.TypeOf(zeroStringer)
)

type simpleString string

func (s simpleString) String() string { return string(s) }

var zeroSimpleString simpleString

func makeStringer(s interface{}) Stringer {
    switch s.(type) {
    case string:
        return simpleString(s.(string))
    case Stringer:
        return s.(Stringer)
    default:
        panic("Value must be type 'string' or 'Stringer'")
    }
    return zeroStringer
}

type ListMode uint

const (
    ColumnsAcross ListMode = iota
    ColumnsDown
    Inline
    Rows
)

func List(items interface{}, mode ListMode, option interface{}) {
    ival := reflect.ValueOf(items)
    itype := ival.Type()
    if k := itype.Kind(); k != reflect.Slice {
        panic(os.NewError("List given non-Slice types."))
    }
    strs := make([]string, ival.Len())
    for i := range strs {
        v := ival.Index(i).Interface()
        switch v.(type) {
        case Stringer:
            strs[i] = v.(Stringer).String()
        case string:
            strs[i] = v.(string)
        default:
            panic(os.NewError("List items contain non-string, non-Stringer item"))
        }
    }
    switch mode {
    case ColumnsAcross:
        fallthrough
    case ColumnsDown:
        wrap := 80
        switch option.(type) {
        case nil:
        case int:
            wrap = option.(int)
        default:
            panic(os.NewError("List option of unacceptable type"))
        }

        var width int
        for i := range strs {
            if n := len(strs[i]); n > width {
                width = n
            }
        }

        n := len(strs)
        ncols := (wrap + 1) / (width + 1)

        if ncols <= 1 {
            // Just print rows if no more than 1 column fits.
            for i := range strs {
                SayTrimmed(strs[i])
            }
            break
        }

        nrows := (n + ncols - 1) / ncols

        sfmt := fmt.Sprintf("%%-%ds", width)
        for i := range strs {
            strs[i] = fmt.Sprintf(sfmt, strs[i])
        }

        switch mode {
        case ColumnsAcross:
            for i := 0; i < n; i += ncols {
                end := i + ncols
                if end > n {
                    end = n
                }
                row := strs[i:end]
                SayTrimmed(strings.Join(row, " "))
            }
        case ColumnsDown:
            for i := 0; i < nrows; i++ {
                var row []string
                for j := 0; j < ncols; j++ {
                    index := j*nrows + i
                    if index >= n {
                        break
                    }
                    row = append(row, strs[index])
                }
                SayTrimmed(strings.Join(row, " "))
            }
        }
    case Inline:
        n := len(strs)
        if n == 1 {
            SayTrimmed(strs[0])
            break
        }
        join := " or "
        switch option.(type) {
        case nil:
        case string:
            join = option.(string)
        default:
            panic(os.NewError("List option of unacceptable type"))
        }
        if n == 2 {
            Say(strings.Join([]string{strs[n-2], join, strs[n-2], "\n"}, ""))
            break
        }
        strs[n-1] = join + strs[n-1]
        SayTrimmed(strings.Join(strs, ", "))
    case Rows:
        for i := range strs {
            SayTrimmed(strs[i])
        }
    default:
        panic(os.NewError("Unknown mode"))
    }
}

//  Prompt the user for text input. The result is stored in dest, which must
//  be a pointer to a native Go type (int, uint16, string, float32, ...).
//  Slice types are not currently supported. List input must be done with a
//  *string destination and post-processing.
func Ask(dest interface{}, msg string, config func(*Question)) (e os.Error) {
    var q *Question
    defer func() {
        if err := recover(); err != nil {
            switch err.(type) {
            case os.Error:
                // Call a panic method...
                if q.Panic != nil {
                    q.Panic(err.(os.Error))
                }
            default:
                panic(err)
            }
        }
    }()
    if k := reflect.TypeOf(dest).Kind(); k != reflect.Ptr && k != reflect.Slice {
        panicUnrecoverable(fmt.Errorf("Ask(...) requires a Ptr type, not %s", k.String()))
        return
    } else if k == reflect.Slice {
        panicUnrecoverable(fmt.Errorf("Ask(...) can not currently assign to slices."))
        return
    }

    var t Type
    switch dest.(type) {
    case *uint:
        t = Uint
    case *uint8:
        t = Uint
    case *uint16:
        t = Uint
    case *uint32:
        t = Uint
    case *uint64:
        t = Uint
    case *int:
        t = Int
    case *int8:
        t = Int
    case *int16:
        t = Int
    case *int32:
        t = Int
    case *int64:
        t = Int
    case *float32:
        t = Float
    case *float64:
        t = Float
    case *string:
        t = String
    default:
        fmt.Errorf("Unusable destination")
    }
    q = newQuestion(t)
    q.Question = msg
    if config != nil {
        config(q)
    }

    if err := q.tryFirstAnswer(); err == nil && q.val != nil {
        if err := q.setDest(dest); err != nil {
            panicUnrecoverable(err)
            q.val = nil
        }
        return
    }

    prompt := msg
    contFunc := func(err os.Error) {
        Say(fmt.Sprintf("Error: %s\n", err.String()))
        prompt = q.Responses[AskOnError]
    }
    r := bufio.NewReader(os.Stdin)
    for {
        tail := stringSuffixFunc(prompt, unicode.IsSpace)
        Say(prompt + q.defaultString(tail))
        var resp []byte
        for cont := true; cont; {
            s, isPrefix, err := r.ReadLine()
            if err != nil {
                panicUnrecoverable(err)
                return
            }
            resp = append(resp, s...)
            cont = isPrefix
        }
        if err := q.parse(string(resp)); err != nil {
            panicUnrecoverable(err)
            contFunc(err)
            continue
        }

        // Cast the result from a wide (e.g. 64bit) type to the desired type.
        // This should not fail under any normal circumstances, so failure
        // should break the loop.
        if err := q.setDest(dest); err != nil {
            panicUnrecoverable(err)
            contFunc(err)
            continue
        }
        break
    }
    return
}

func Confirm(question string, yes bool, config func(*Question)) bool {
    def := "no"
    if yes {
        def = "yes"
    }

    var okstr string
    var err os.Error
    Ask(&okstr, question, func(q *Question) {
        q.Default = def
        q.In(StringSet{"yes", "y", "no", "n"})
        if config != nil {
            config(q)
        }
        if q.Panic != nil {
            f := q.Panic
            q.Panic = func(e os.Error) {
                err = e
                f(e)
            }
        }
    })
    if err != nil {
        return false
    }
    if okstr[0] == 'y' {
        return true
    }
    return false
}
