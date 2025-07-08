package list

import (
	"fmt"
	"sync"
)

// List Definir la interfaz List
type List[T any] interface {
	Add(item T)                                          // Añadir un elemento al final de la lista
	Dequeue() (T, error)                                 // Eliminar y devolver el primer elemento de la lista
	Filter(value T, predicate func(a, b T) bool) List[T] // Filtra elementos de la lista
	Find(predicate func(T) bool) (T, int, bool)          // Permite buscar un elemento de la lista dado un predicado.
	FindAll(predicate func(T) bool) *ArrayList[T]        // FindAll encuentra todos los elementos que satisfacen el predicado y los devuelve en una nueva lista
	ForEach(callback func(T))                            // A cada elemento de la lista se le va aplicar la función que le pase
	Get(index int) (T, error)                            // Obtener un elemento a partir de un índice dado
	GetAll() []T                                         // Retorna todos los elementos que se encuentra en la lista
	Insert(index int, item T) error                      // Insertar un elemento en el índice dado
	Pop() (T, error)                                     // Remover el último elemento de la lista
	Remove(index int)                                    // Eliminar un elemento en el índice dado
	RemoveWhere(match func(T) bool)
	Set(index int, newValue T) error // Modifica el valor de un elemento de la lista a partir de su índice.
	Size() int                       // Retornar el tamaño de la lista
	Sort(less func(a, b T) bool)     // Ordena una Lista de acuerdo al criterio
}

// ArrayList implements List
type ArrayList[T any] struct {
	mu    sync.RWMutex
	items []T
}

// Add inserta un elemento al final de la lista.
//
// Parámetros:
//   - item: Elemento a insertar.
//
// Ejemplo:
//
//	func main() {
//		list := &ArrayList[int]{}
//		list.Add(10)
//		list.Add(20)
//	}
func (list *ArrayList[T]) Add(item T) {
	list.mu.Lock() // Bloqueo exclusivo para evitar cambios simultáneos
	defer list.mu.Unlock()

	list.items = append(list.items, item)
}

// Dequeue elimina y devuelve el primer elemento de la cola.
// En caso de que la lista se encuentre vacío retorna el valor "cero" del tipo T y un error indicando que está vacía.
//
// Ejemplo:
//
//	func main() {
//		numbers := &list.ArrayList[int]{}
//		numbers.Add(10)
//		numbers.Add(20)
//		numbers.Add(30)
//		value, _ := numbers.Dequeue()
//		fmt.Println("Valor: ", value) //output: 10
//	}
func (list *ArrayList[T]) Dequeue() (T, error) {
	list.mu.Lock() // Bloqueo exclusivo para evitar cambios simultáneos
	defer list.mu.Unlock()

	if len(list.items) == 0 {
		var zero T // Devuelve el valor "cero" del tipo T
		return zero, fmt.Errorf("list is empty")
	}
	valor := list.items[0]
	list.items = list.items[1:]
	return valor, nil
}

// Filter filtra elementos de la lista a partir de un predicado.
//
// Parámetros:
//   - value: Valor a comparar.
//   - predicate: Función que compara dos elementos.
//
// Ejemplo:
//
//	func main() {
//		list := &ArrayList[int]{}
//		list.Add(10)
//		list.Add(20)
//		list.Add(30)
//		list.Add(40)
//		predicate := func(a int, b int) bool {
//			return b > a
//		}
//		//en este caso devuelva una nueva lista con aquellos que son mayores que 20
//		filtered := list.Filter(20, predicate)
//	}
func (list *ArrayList[T]) Filter(value T, predicate func(a, b T) bool) List[T] {
	list.mu.Lock() // Bloqueo exclusivo para evitar cambios simultáneos
	defer list.mu.Unlock()

	filteredList := &ArrayList[T]{}

	for _, item := range list.items {
		if predicate(value, item) {
			filteredList.Add(item)
		}
	}

	return filteredList
}

// Find permite buscar un elemento de la lista dado un predicado.
//
// Parámetros:
//   - predicate: Función que permite identificar el elemento buscado.
//
// Ejemplo:
//
//	func main() {
//		list := &ArrayList[int]{}
//
//		list.Add(10)
//		list.Add(20)
//		list.Add(30)
//
//		number, found := list.Find(func(number int) bool {
//			return number == 20
//		})
//	}
func (list *ArrayList[T]) Find(predicate func(T) bool) (T, int, bool) {
	list.mu.RLock() //Bloqueo de solo lectura: permite otras lecturas concurrentes
	defer list.mu.RUnlock()

	for i, item := range list.items {
		if predicate(item) {
			return item, i, true
		}
	}
	var zero T
	return zero, -1, false
}

// NewArrayList crea y devuelve una nueva instancia de ArrayList.
func newArrayList[T any]() *ArrayList[T] {
	return &ArrayList[T]{
		items: make([]T, 0), // Inicializa el slice interno vacío
	}
}

// FindAll encuentra todos los elementos que satisfacen el predicado y los devuelve en una nueva instancia de ArrayList.
// Si ningún elemento cumple la condición, devuelve un ArrayList vacío.
func (list *ArrayList[T]) FindAll(predicate func(T) bool) *ArrayList[T] {
	list.mu.RLock() //Bloqueo de solo lectura: permite otras lecturas concurrentes
	defer list.mu.RUnlock()

	// Crea una nueva instancia de ArrayList para almacenar los elementos filtrados
	filteredList := newArrayList[T]()

	for _, item := range list.items {
		if predicate(item) {
			filteredList.Add(item)
		}
	}

	return filteredList
}

// Get devuelve el elemento en el índice proporcionado.
//
// Parámetros:
//   - index: Índice del elemento a obtener.
//
// Ejemplo:
//
//	func main() {
//		list := &ArrayList[int]{}
//		list.Add(10)
//		list.Add(20)
//		list.Add(30)
//
//		value, _ := list.Get(1)
//		fmt.Println("Valor: ", value) //Output: 20
//	}
func (list *ArrayList[T]) Get(index int) (T, error) {
	list.mu.RLock() //Bloqueo de solo lectura: permite otras lecturas concurrentes
	defer list.mu.RUnlock()

	// Validar si el índice está dentro del rango
	if index < 0 || index >= len(list.items) {
		// Get item from a List
		var zero T // Crear un valor cero del tipo genérico T
		return zero, fmt.Errorf("index out of range: %d", index)
	}
	return list.items[index], nil
}

// Insert inserta un elemento en la lista en el índice proporcionado.
//
// Parámetros:
//   - index: Índice donde se va a ingresar el elemento.
//   - item: Elemento a ingresar.
//
// Ejemplo:
//
//	func main() {
//		list := &ArrayList[int]{}
//		list.Add(10)
//		list.Add(30)
//
//		_ := list.Insert(1, 100) [10, 100, 30]
//	}
func (list *ArrayList[T]) Insert(index int, item T) error {
	list.mu.Lock() // Bloqueo exclusivo para evitar cambios simultáneos
	defer list.mu.Unlock()

	if index < 0 || index > len(list.items) {
		return fmt.Errorf("index out of range: %d", index)
	}
	list.items = append(list.items[:index], append([]T{item}, list.items[index:]...)...)
	return nil
}

// Pop remueve el último elemento de la lista y lo devuelve.
//
// Parámetros:
//   - value: Valor a comparar.
//   - predicate: Función que compara dos elementos.
//
// Ejemplo:
//
//	func main() {
//		list := &ArrayList[int]{}
//		list.Add(10)
//		list.Add(20)
//		list.Add(30)
//
//		value, _ := list.Pop()
//		fmt.Println("Valor: ", value) //Output: 30
//	}
func (list *ArrayList[T]) Pop() (T, error) {
	list.mu.Lock() // Bloqueo exclusivo para evitar cambios simultáneos
	defer list.mu.Unlock()

	if len(list.items) == 0 {
		var zero T
		return zero, fmt.Errorf("list is empty")
	}
	lastIndex := len(list.items) - 1
	item := list.items[lastIndex]
	list.items = list.items[:lastIndex]
	return item, nil
}

// Remove remueve un elemento de la lista a partir de su índice.
//
// Parámetros:
//   - list: lista de cualquier tipo.
//   - index: Índice del elemento a remover.
//
// Ejemplo:
//
//	func main() {
//		list := &ArrayList[int]{}
//		list.Add(10)
//		list.Add(20)
//		list.Add(30)
//		// Eliminar el elemento en índice 1
//		list.Remove(1)  //[10, 30]
//	}
func (list *ArrayList[T]) Remove(index int) {
	list.mu.Lock() // Bloqueo exclusivo para evitar cambios simultáneos
	defer list.mu.Unlock()

	if index >= 0 && index < len(list.items) {
		list.items = append(list.items[:index], list.items[index+1:]...)
	}
}

func (list *ArrayList[T]) RemoveWhere(match func(T) bool) {
	list.mu.Lock() // Bloqueo exclusivo para evitar cambios simultáneos
	defer list.mu.Unlock()

	for i, item := range list.items {
		if match(item) {
			list.items = append(list.items[:i], list.items[i+1:]...)
			break
		}
	}
}

// Set modifica el valor de un elemento de la lista a partir de su índice.
//
// Parámetros:
//   - list: lista de cualquier tipo.
//   - index: Índice del elemento a remover.
//
// Ejemplo:
//
//	type Person struct {
//		id   int
//		name string
//		mail string
//	}
//	func main() {
//		persons = ArrayList[Person]{}
//		persons.Add(Person{id: 1, name: "pepe", mail: "pepe@mail.com"})
//
//		person.name = "test"
//		_ = persons.Set(index, person) //Person{id: 1, name: "test", mail: "pepe@mail.com"}
//	}
func (list *ArrayList[T]) Set(index int, newValue T) error {
	list.mu.Lock() // Bloqueo exclusivo para evitar cambios simultáneos
	defer list.mu.Unlock()

	if index < 0 || index >= len(list.items) {
		return fmt.Errorf("index out of range:  %d", index)
	}
	list.items[index] = newValue
	return nil
}

// Size devuelve el tamaño de la lista.
//
// Ejemplo
//
//	func main() {
//	    list := &ArrayList[int]{}
//
//	    list.Add(10)
//	    list.Add(20)
//	    list.Add(30)
//
//	    size := list.Size()
//	    fmt.Println("Size: ", size) //output: 3
//	}
func (list *ArrayList[T]) Size() int {
	list.mu.RLock() ///Bloqueo de solo lectura: permite otras lecturas concurrentes
	defer list.mu.RUnlock()

	return len(list.items)
}

// Sort ordena una lista de acuerdo a un criterio.
//
// Parámetros:
//   - value: Valor a comparar.
//   - predicate: Función que compara dos elementos.
//
// Ejemplo:
//
//	func main() {
//		list := &ArrayList[int]{}
//		list.Add(40)
//		list.Add(20)
//		list.Add(30)
//		list.Add(10)
//
//		predicate := func(a int, b int) bool {
//			return a < b
//		}
//
//		list.Sort(predicate) //[10, 20, 30, 40]
//	}
func (list *ArrayList[T]) Sort(less func(a, b T) bool) {
	list.mu.Lock() // Bloqueo exclusivo para evitar cambios simultáneos
	defer list.mu.Unlock()

	size := list.Size()
	for i := 0; i < size-1; i++ {
		for j := 0; j < size-i-1; j++ {
			if !less(list.items[j], list.items[j+1]) {
				// Intercambiar los elementos si están en el orden incorrecto
				list.items[j], list.items[j+1] = list.items[j+1], list.items[j]
			}
		}
	}
}

// ForEach a cada elemento de la lista se va a aplicar la función que le pase.
//
// Parámetros:
//   - callback: es una función que se ejecuta para cada elemento de la lista.
//
// Ejemplo
//
//	func main() {
//	    list := &ArrayList[int]{}
//
//	    list.Add(10)
//	    list.Add(20)
//	    list.Add(30)
//
//	    list.ForEach(func(number int) {
//	    	fmt.Println("Valor:", number)
//	    })
//	}
func (list *ArrayList[T]) ForEach(callback func(T)) {
	list.mu.Lock() // Bloqueo exclusivo para evitar cambios simultáneos
	defer list.mu.Unlock()

	for _, item := range list.items {
		callback(item)
	}
}

// GetAll retorna una copia de todos los elementos que se encuentra en la lista
func (list *ArrayList[T]) GetAll() []T {
	list.mu.RLock() //Bloqueo de solo lectura: permite otras lecturas concurrentes
	defer list.mu.RUnlock()

	//list.mu.Lock() // Bloquear el mutex
	//defer list.mu.Unlock() // Asegurar que se libere

	// Crear una copia del slice para evitar que modificaciones externas afecten la lista interna
	itemsCopy := make([]T, len(list.items))
	copy(itemsCopy, list.items)
	return itemsCopy
}
