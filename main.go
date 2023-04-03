package main

// import (
// 	"errors"
// 	"fmt"
// 	"math/rand"
// 	"sync"
// )

// type service struct {
// 	Service map[string]interface{}
// 	Mu      sync.Mutex
// }

// func main() {
// 	wg := sync.WaitGroup{}
// 	wg.Add(3)
// 	s := service{
// 		Service: map[string]interface{}{},
// 		Mu:      sync.Mutex{},
// 	}

// 	go func(s *service) {
// 		for {
//
// 			set(fmt.Sprint(rand.Intn(10)), rand.Int(), s)
// 			v, err := get(fmt.Sprint(rand.Intn(10)), s)
// 			if err != nil {
// 				fmt.Println(err)
// 			} else {
// 				fmt.Println(v)
// 			}
// 		}
// 	}(&s)

// 	go func(s *service) {
// 		for {

// 			set(fmt.Sprint(rand.Intn(10)), rand.Int(), s)
// 			v, err := get(fmt.Sprint(rand.Intn(10)), s)
// 			if err != nil {
// 				fmt.Println(err)
// 			} else {
// 				fmt.Println(v)
// 			}
// 		}
// 	}(&s)

// 	go func(s *service) {
// 		for {

// 			set(fmt.Sprint(rand.Intn(10)), rand.Int(), s)
// 			v, err := get(fmt.Sprint(rand.Intn(10)), s)
// 			if err != nil {
// 				fmt.Println(err)
// 			} else {
// 				fmt.Println(v)
// 			}
// 		}
// 	}(&s)

// 	go func(s *service) {
// 		for {

// 			set(fmt.Sprint(rand.Intn(10)), rand.Int(), s)
// 			v, err := get(fmt.Sprint(rand.Intn(10)), s)
// 			if err != nil {
// 				fmt.Println(err)
// 			} else {
// 				fmt.Println(v)
// 			}
// 		}
// 	}(&s)

// 	go func(s *service) {
// 		for {

// 			set(fmt.Sprint(rand.Intn(10)), rand.Int(), s)
// 			v, err := get(fmt.Sprint(rand.Intn(10)), s)
// 			if err != nil {
// 				fmt.Println(err)
// 			} else {
// 				fmt.Println(v)
// 			}
// 		}
// 	}(&s)

// 	go func(s *service) {
// 		for {

// 			set(fmt.Sprint(rand.Intn(10)), rand.Int(), s)
// 			v, err := get(fmt.Sprint(rand.Intn(10)), s)
// 			if err != nil {
// 				fmt.Println(err)
// 			} else {
// 				fmt.Println(v)
// 			}
// 		}
// 	}(&s)

// 	go func(s *service) {
// 		for {

// 			set(fmt.Sprint(rand.Intn(10)), rand.Int(), s)
// 			v, err := get(fmt.Sprint(rand.Intn(10)), s)
// 			if err != nil {
// 				fmt.Println(err)
// 			} else {
// 				fmt.Println(v)
// 			}
// 		}
// 	}(&s)

// 	go func(s *service) {
// 		for {

// 			set(fmt.Sprint(rand.Intn(10)), rand.Int(), s)
// 			v, err := get(fmt.Sprint(rand.Intn(10)), s)
// 			if err != nil {
// 				fmt.Println(err)
// 			} else {
// 				fmt.Println(v)
// 			}
// 		}
// 	}(&s)

// 	go func(s *service) {
// 		for {

// 			set(fmt.Sprint(rand.Intn(10)), rand.Int(), s)
// 			v, err := get(fmt.Sprint(rand.Intn(10)), s)
// 			if err != nil {
// 				fmt.Println(err)
// 			} else {
// 				fmt.Println(v)
// 			}
// 		}
// 	}(&s)

// 	wg.Wait()
// }

// func get(name string, s *service) (interface{}, error) {
// 	if s == nil {
// 		return nil, errors.New("error get")
// 	}

// 	s.Mu.Lock()
// 	defer s.Mu.Unlock()

// 	val, ok := s.Service[name]
// 	if !ok {
// 		return nil, errors.New("err get, not ok")
// 	}

// 	return val, nil
// }

// func set(name string, val interface{}, s *service) error {
// 	if s == nil {
// 		return errors.New("error set")
// 	}

// 	s.Mu.Lock()
// 	defer s.Mu.Unlock()

// 	_, ok := s.Service[name]
// 	if ok {
// 		return errors.New("error set, already exists")
// 	}

// 	s.Service[name] = val

// 	return nil
// }
