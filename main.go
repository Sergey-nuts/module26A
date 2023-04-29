package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var RingBuffSize int = 5                      // размер кольцевого буфера
var Duration time.Duration = time.Second * 10 // интервал опустошения буфера

// Кольцевой буфер
type RingBuff struct {
	m    sync.Mutex // Mutex для блокировки буфера
	data []int      // данные
	p    int        // позиция (-1 в случае пустого буфера)
	size int        // размер буфера
}

// Создание нового экземпляра кольцевого буфера размером n
func NewRingBuff(n int) *RingBuff {
	return &RingBuff{sync.Mutex{}, make([]int, n), -1, n}
}

// добавление елемента el в кольцевой буфер
// если буфер был заполнен, то отбрасывается самое старое значение
func (r *RingBuff) Push(el int) {
	r.m.Lock()
	defer r.m.Unlock()
	if r.p == r.size-1 {
		for i := 1; i < r.size; i++ {
			r.data[i-1] = r.data[i]
		}
		r.p--
	}
	r.p++
	r.data[r.p] = el
}

// Опустошение кольцевого буфера
// возвращает слайс значений кольцевого буфера
func (r *RingBuff) Get() []int {
	if r.p < 0 {
		return nil
	}
	r.m.Lock()
	defer r.m.Unlock()
	res := make([]int, r.p+1)
	for i := 0; i <= r.p; i++ {
		res[i] = r.data[i]
	}
	r.p = -1
	return res
}

// Инициализация пайплайна
func makePipeline(input chan int) chan bool {
	done := make(chan bool)
	notNegInt := make(chan int)
	divThree := make(chan int)
	buff := NewRingBuff(RingBuffSize)

	notNegPipe(done, input, notNegInt)
	divThreePipe(done, notNegInt, divThree)
	buffPipe(done, divThree, buff)

	return done
}

// Первая стадия пайплайна.
// Фильтрация отрицатьельных чисел из канала in в канал out
func notNegPipe(done chan bool, in <-chan int, out chan<- int) {
	go func() {
		for {
			select {
			case d := <-in:
				if d >= 0 {
					out <- d
				}
			case <-done:
				return
			}
		}
	}()
}

// Вторая стадия пайплайна
// Фильтрация чисел не кратных 3 из канала in в канал out
func divThreePipe(done chan bool, in <-chan int, out chan<- int) {
	go func() {
		for {
			select {
			case d := <-in:
				if d%3 == 0 && d != 0 {
					out <- d
				}
			case <-done:
				return
			}
		}
	}()
}

// Третья стадия пайплайна
// Буферизация данных из канала in в кольцевой буфер buff
func buffPipe(done chan bool, in chan int, buff *RingBuff) {
	go func() {
		for {
			select {
			case d := <-in:
				buff.Push(d)
			case <-done:
				return
			}
		}
	}()
	consumer(done, buff)
}

// С интервалом Duration опустошает буфер buff
// выводит в консоль содержимое буфера, если он был не пустой
func consumer(done chan bool, buff *RingBuff) {
	var ticker *time.Ticker = time.NewTicker(Duration)
	go func() {
		for {
			select {
			case <-ticker.C:
				res := buff.Get()
				if len(res) > 0 {
					fmt.Printf("result data: %v\n", res)
				}
			case <-done:
				return
			}
		}
	}()
}

func main() {
	input := make(chan int)
	done := makePipeline(input)

	scanner := bufio.NewScanner(os.Stdin)
	var data string
	for {
		scanner.Scan()
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
			close(done)
			break
		}
		data = scanner.Text()
		if strings.EqualFold(data, "exit") {
			fmt.Printf("closing program\n")
			close(done)
			break
		}
		d, err := strconv.Atoi(data)
		if err != nil {
			fmt.Printf("input only digits or `exit`\n")
			continue
		}
		input <- d
	}
}
