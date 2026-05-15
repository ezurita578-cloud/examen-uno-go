package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

// Card representa una carta del juego
type Card struct {
	Color string
	Value string
}

// Player representa un jugador con su nombre y sus cartas
type Player struct {
	Name string
	Hand []Card
}

// Game tiene todo el estado del juego
type Game struct {
	Deck        []Card
	Players     []Player
	DiscardPile []Card
	Turn        int
	Direction   int // 1 = normal, -1 = reversa
}

// mostrarCarta regresa un string con el color y valor de la carta
func mostrarCarta(c Card) string {
	return "[" + c.Color + " " + c.Value + "]"
}

// esEspecial revisa si la carta tiene un efecto especial
func esEspecial(c Card) bool {
	if c.Value == "+2" || c.Value == "Reversa" || c.Value == "Salto" {
		return true
	}
	return false
}

// esJugable revisa si una carta se puede jugar sobre la carta en mesa
func esJugable(carta Card, enMesa Card) bool {
	if carta.Color == enMesa.Color {
		return true
	}
	if carta.Value == enMesa.Value {
		return true
	}
	return false
}

// iniciarMazo crea todas las cartas y baraja el mazo
func (g *Game) iniciarMazo() {
	colores := []string{"Rojo", "Verde", "Azul", "Amarillo"}
	numeros := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	// punto extra: cartas especiales
	especiales := []string{"+2", "Reversa", "Salto"}

	for _, color := range colores {
		for _, numero := range numeros {
			g.Deck = append(g.Deck, Card{Color: color, Value: numero})
		}
		// agrego las especiales por color
		for _, especial := range especiales {
			g.Deck = append(g.Deck, Card{Color: color, Value: especial})
		}
	}

	// barajar el mazo
	rand.Shuffle(len(g.Deck), func(i, j int) {
		g.Deck[i], g.Deck[j] = g.Deck[j], g.Deck[i]
	})
}

// tomarCarta saca la primera carta del mazo
func (g *Game) tomarCarta() Card {
	// si el mazo se acaba, recyclar el descarte
	if len(g.Deck) == 0 {
		fmt.Println("El mazo se acabó, reciclando descarte...")
		ultima := g.DiscardPile[len(g.DiscardPile)-1]
		g.Deck = g.DiscardPile[:len(g.DiscardPile)-1]
		g.DiscardPile = []Card{ultima}
		rand.Shuffle(len(g.Deck), func(i, j int) {
			g.Deck[i], g.Deck[j] = g.Deck[j], g.Deck[i]
		})
	}
	carta := g.Deck[0]
	g.Deck = g.Deck[1:]
	return carta
}

// repartirCartas da 5 cartas a cada jugador
func (g *Game) repartirCartas() {
	for i := 0; i < len(g.Players); i++ {
		for j := 0; j < 5; j++ {
			carta := g.tomarCarta()
			g.Players[i].Hand = append(g.Players[i].Hand, carta)
		}
	}
	// voltear primera carta para empezar (que no sea especial)
	for {
		primera := g.tomarCarta()
		if !esEspecial(primera) {
			g.DiscardPile = append(g.DiscardPile, primera)
			break
		}
		g.Deck = append(g.Deck, primera)
	}
}

// siguienteTurno avanza al siguiente jugador
func (g *Game) siguienteTurno() {
	n := len(g.Players)
	g.Turn = (g.Turn + g.Direction + n) % n
}

// aplicarEspecial aplica el efecto de una carta especial
// (esto es el punto extra, lo intenté implementar)
func (g *Game) aplicarEspecial(carta Card, mensajes chan string) {
	switch carta.Value {
	case "+2":
		g.siguienteTurno()
		victima := &g.Players[g.Turn]
		c1 := g.tomarCarta()
		c2 := g.tomarCarta()
		victima.Hand = append(victima.Hand, c1, c2)
		mensajes <- ">> " + victima.Name + " roba 2 cartas y pierde su turno!"
	case "Reversa":
		g.Direction = g.Direction * -1
		mensajes <- ">> Direccion invertida!"
	case "Salto":
		g.siguienteTurno()
		mensajes <- ">> Turno saltado!"
	}
}

// jugarTurno maneja el turno de un jugador
// mensajes es un canal para mandar texto a la goroutine que imprime
func (g *Game) jugarTurno(mensajes chan string) bool {
	jugador := &g.Players[g.Turn]
	enMesa := g.DiscardPile[len(g.DiscardPile)-1]
	reader := bufio.NewReader(os.Stdin)

	mensajes <- "\n================================"
	mensajes <- "Turno de: " + jugador.Name
	mensajes <- "Carta en mesa: " + mostrarCarta(enMesa)
	mensajes <- "================================"

	// mostrar cartas del jugador
	fmt.Println("\nTu mano:")
	jugables := []int{}
	for i := 0; i < len(jugador.Hand); i++ {
		carta := jugador.Hand[i]
		marca := "  "
		if esJugable(carta, enMesa) {
			marca = "* "
			jugables = append(jugables, i)
		}
		fmt.Printf("  %s%d) %s\n", marca, i+1, mostrarCarta(carta))
	}

	// si no tiene jugables, robar obligatorio
	if len(jugables) == 0 {
		mensajes <- "No tienes cartas jugables, robas una carta..."
		robada := g.tomarCarta()
		jugador.Hand = append(jugador.Hand, robada)
		mensajes <- "Robaste: " + mostrarCarta(robada)

		if esJugable(robada, enMesa) {
			mensajes <- "La carta robada es jugable! Se juega automaticamente."
			g.DiscardPile = append(g.DiscardPile, robada)
			jugador.Hand = jugador.Hand[:len(jugador.Hand)-1]
			if esEspecial(robada) {
				g.aplicarEspecial(robada, mensajes)
			}
		}
		return false
	}

	// pedir eleccion
	fmt.Printf("\nElige carta (1-%d) o 0 para robar: ", len(jugador.Hand))
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	eleccion, err := strconv.Atoi(input)

	// si ingresa algo invalido, robar
	if err != nil || eleccion < 0 || eleccion > len(jugador.Hand) {
		mensajes <- "Entrada invalida, robas una carta."
		robada := g.tomarCarta()
		jugador.Hand = append(jugador.Hand, robada)
		mensajes <- "Robaste: " + mostrarCarta(robada)
		return false
	}

	// robar voluntario
	if eleccion == 0 {
		robada := g.tomarCarta()
		jugador.Hand = append(jugador.Hand, robada)
		mensajes <- "Robaste: " + mostrarCarta(robada)
		return false
	}

	// revisar que la carta elegida sea valida
	elegida := jugador.Hand[eleccion-1]
	if !esJugable(elegida, enMesa) {
		mensajes <- "Esa carta no es valida! Intenta de nuevo."
		return g.jugarTurno(mensajes)
	}

	// jugar la carta
	g.DiscardPile = append(g.DiscardPile, elegida)
	// quitar la carta de la mano
	jugador.Hand = append(jugador.Hand[:eleccion-1], jugador.Hand[eleccion:]...)
	mensajes <- jugador.Name + " jugo " + mostrarCarta(elegida)

	// revisar si gano
	if len(jugador.Hand) == 0 {
		mensajes <- "*** " + jugador.Name + " ha ganado! ***"
		return true
	}

	// avisar UNO
	if len(jugador.Hand) == 1 {
		mensajes <- ">> " + jugador.Name + " dice UNO!"
	}

	// aplicar efecto si es especial
	if esEspecial(elegida) {
		g.aplicarEspecial(elegida, mensajes)
	}

	return false
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("╔══════════════════════╗")
	fmt.Println("║    JUEGO DE UNO      ║")
	fmt.Println("╚══════════════════════╝")

	// pedir nombres
	fmt.Print("Nombre Jugador 1: ")
	nombre1, _ := reader.ReadString('\n')
	nombre1 = strings.TrimSpace(nombre1)
	if nombre1 == "" {
		nombre1 = "Jugador 1"
	}

	fmt.Print("Nombre Jugador 2: ")
	nombre2, _ := reader.ReadString('\n')
	nombre2 = strings.TrimSpace(nombre2)
	if nombre2 == "" {
		nombre2 = "Jugador 2"
	}

	// crear el juego
	juego := Game{
		Players: []Player{
			{Name: nombre1},
			{Name: nombre2},
		},
		Direction: 1,
	}

	juego.iniciarMazo()
	juego.repartirCartas()

	fmt.Println("\nCarta inicial:", mostrarCarta(juego.DiscardPile[0]))
	fmt.Println("Que comience el juego!")

	// canal de mensajes para la goroutine que imprime
	mensajes := make(chan string)

	// goroutine que escucha el canal e imprime los mensajes
	go func() {
		for msg := range mensajes {
			fmt.Println(msg)
		}
	}()

	// bucle principal del juego
	for {
		gano := juego.jugarTurno(mensajes)
		if gano {
			break
		}
		juego.siguienteTurno()
	}

	// cerrar el canal cuando termina el juego
	close(mensajes)
	fmt.Println("\nGracias por jugar!")
}
