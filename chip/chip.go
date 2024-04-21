package chip

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"

	//"strings"
	"bufio"
	"math/rand"
	"os"
	"strconv"
	"time"
)

const SHOW_DISASM = false
const DEBUG = false
const O_HLT_ON_LOOP = true
const SCREEN_WIDTH = 64
const SCREEN_HEIGHT = 32
const STACK_BASE = 0x200
const HEXDIGIT_SPRITES_BASE = 0x0000
const START_ADDR = 0x200

var HEXDIGIT_SPRITES = []byte{
	0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
	0x20, 0x60, 0x20, 0x20, 0x70, // 1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
	0x90, 0x90, 0xF0, 0x10, 0x10, // 4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
	0xF0, 0x10, 0x20, 0x40, 0x40, // 7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
	0xF0, 0x90, 0xF0, 0x90, 0x90, // A
	0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
	0xF0, 0x80, 0x80, 0x80, 0xF0, // C
	0xE0, 0x90, 0x90, 0x90, 0xE0, // D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
	0xF0, 0x80, 0xF0, 0x80, 0x80, // F
}

type Instruction_kind string

const (
	CLS         Instruction_kind = "CLS"
	RET                          = "RET"
	JMP                          = "JMP"
	CALL_NNN                     = "CALL_NNN"
	SE_VX_NN                     = "SE_VX_NN"
	SNE_VX_NN                    = "SNE_VX_NN"
	SE_VX_VY                     = "SE_VX_VY"
	LD_VX_NN                     = "LD_VX_NN"
	ADD_VX_NN                    = "ADD_VX_NN"
	LD_VX_VY                     = "LD_VX_VY"
	OR_VX_VY                     = "OR_VX_VY"
	AND_VX_VY                    = "AND_VX_VY"
	XOR_VX_VY                    = "XOR_VX_VY"
	ADD_VX_VY                    = "ADD_VX_VY"
	SUB_VX_VY                    = "SUB_VX_VY"
	SHR_VX_VY                    = "SHR_VX_VY"
	SUBN_VX_VY                   = "SUBN_VX_VY"
	SHL_VX_VY                    = "SHL_VX_VY"
	SNE_VX_VY                    = "SNE_VX_VY"
	LD_I_NNN                     = "LD_I_NNN"
	JMP_V0_NNN                   = "JMP_V0_NNN"
	RND_VX_NN                    = "RND_VX_NN"
	DRW_VX_VY_N                  = "DRW_VX_VY_N"
	SKP_VX                       = "SKP_VX"
	SKNP_VX                      = "SKNP_VX"
	LD_VX_DT                     = "LD_VX_DT"
	LD_VX_K                      = "LD_VX_K"
	LD_DT_VX                     = "LD_DT_VX"
	LD_ST_VX                     = "LD_ST_VX"
	ADD_I_VX                     = "ADD_I_VX"
	LD_F_VX                      = "LD_F_VX"
	LD_B_VX                      = "LD_B_VX"
	LD_I_VX                      = "LD_I_VX"
	LD_VX_I                      = "LD_VX_I"
	INVALID                      = "INVALID"
)

type Instruction struct {
	kind    Instruction_kind
	address uint16
	data    uint16
}

func (ins *Instruction) get_group() uint8 {
	return uint8((ins.data & 0xF000) >> 12)
}
func (ins *Instruction) get_x() uint8 {
	return uint8((ins.data & 0x0F00) >> 8)
}
func (ins *Instruction) get_y() uint8 {
	return uint8((ins.data & 0x00F0) >> 4)
}
func (ins *Instruction) get_n() uint8 {
	return uint8(ins.data & 0x000F)
}
func (ins *Instruction) get_nn() uint8 {
	return uint8(ins.data) & 0x00FF
}
func (ins *Instruction) get_nnn() uint16 {
	return ins.data & 0x0FFF
}
func (ins *Instruction) String() string {
	return fmt.Sprintf("0x%x | %04x : %v", ins.address, ins.data, ins.kind)
}

type Chip struct {
	memory  [0x1000]byte
	display [SCREEN_HEIGHT][SCREEN_WIDTH]bool
	pc      uint16
	I       uint16
	dt      uint8
	st      uint8
	sp      uint16
	V       [16]uint8
	redraw  bool
}

func startTimer(chip *Chip) {
	for {
		chip.timer_int()
		chip.Draw()
		time.Sleep(time.Second / 60)
	}
}

func NewChip(rom_path string) *Chip {
	c := new(Chip)
	c.pc = START_ADDR
	c.sp = STACK_BASE
	b, err := ioutil.ReadFile(rom_path)
	if err != nil {
		log.Fatal(err)
	}
	copy(c.memory[HEXDIGIT_SPRITES_BASE:], HEXDIGIT_SPRITES)
	copy(c.memory[c.pc:], b)
	c.redraw = true
	go startTimer(c)
	return c
}

func (chip *Chip) dump() {
	chip.show_regs()
	fmt.Printf("=== memory (code) ===\n")
	chip.show_mem(chip.pc, 30)
	fmt.Printf("=== memory (stack) ===\n")
	chip.show_mem(chip.sp, 30)
	chip.Draw()
}

func (chip *Chip) show_regs() {
	fmt.Printf("   === registers ===\n")
	fmt.Printf("pc = 0x%x\n", chip.pc)
	fmt.Printf("I  = 0x%x\n", chip.I)
	fmt.Printf("dt = 0x%x\n", chip.dt)
	fmt.Printf("st = 0x%x\n", chip.st)
	fmt.Printf("sp = 0x%x\n", chip.sp)
	for i := 0; i < 8; i++ {
		fmt.Printf("V[%x] = 0x%x\tV[%x] = 0x%x\n", i, chip.V[i], i+8, chip.V[i+8])
	}
}

func (chip *Chip) show_mem(base uint16, length uint16) {
	const BYTES_PER_LINE = 16
	var i, j uint16
	for i = 0; i < length; i += BYTES_PER_LINE {
		fmt.Printf("0x%03x | ", base+i)
		for j = 0; j < BYTES_PER_LINE; j++ {
			fmt.Printf("%02x ", chip.memory[base+i+j])
		}
		fmt.Println()
	}
	fmt.Println()
}

func (chip *Chip) Draw() {
	f := bufio.NewWriter(os.Stdout)
	defer f.Flush()
	if chip.redraw {
		f.WriteString("\033[2J")
		str := ""
		for i := -1; i <= len(chip.display); i++ {
			if i == -1 || i == SCREEN_HEIGHT {
				for j := -1; j <= SCREEN_WIDTH; j++ {
					if j == -1 || j == SCREEN_WIDTH {
						if chip.st > 0 {
							str += "#"
							// f.WriteString("#")
						} else {
							str += "+"
							// f.WriteString("+")
						}
					} else {
						str += "──"
						// f.WriteString("──")
					}
				}
				str += "\n"
				// f.WriteString("\n")
				continue
			}
			row := chip.display[i]
			str += "|"
			// f.WriteString("|")
			for _, e := range row {
				if e {
					str += "▓▓"
					// f.WriteString("▓▓")
				} else {
					str += "  "
					// f.WriteString("  ")
				}
			}
			str += "|\n"
			// f.WriteString("|\n")
		}
		f.WriteString(str)
		chip.redraw = false
	}
}

/*
func show_disasm(disasm string) {
    if SHOW_DISASM {
        fmt.Printf("[DISM] %s;\n", disasm)
    }
}
*/

func (chip *Chip) timer_int() {
	if chip.dt > 0 {
		chip.dt--
	}
	if chip.st > 0 {
		chip.st--
	}
}

func (chip *Chip) fetch() uint16 {
	var ret uint16
	ret = uint16(chip.memory[chip.pc]) << 8
	chip.pc++
	ret += uint16(chip.memory[chip.pc])
	chip.pc++
	return ret
}

func (chip *Chip) decode(data uint16) Instruction {
	ins := Instruction{INVALID, chip.pc - 2, data}
	switch ins.get_group() {
	case 0x0:
		if ins.get_x() != 0x0 {
			break
		}
		switch ins.get_nn() {
		case 0xE0:
			ins.kind = CLS
		case 0xEE:
			ins.kind = RET
		}
	case 0x1:
		ins.kind = JMP
	case 0x2:
		ins.kind = CALL_NNN
	case 0x3:
		ins.kind = SE_VX_NN
	case 0x4:
		ins.kind = SNE_VX_NN
	case 0x5:
		ins.kind = SE_VX_VY
	case 0x6:
		ins.kind = LD_VX_NN
	case 0x7:
		ins.kind = ADD_VX_NN
	case 0x8:
		switch ins.get_n() {
		case 0x0:
			ins.kind = LD_VX_VY
		case 0x1:
			ins.kind = OR_VX_VY
		case 0x2:
			ins.kind = AND_VX_VY
		case 0x3:
			ins.kind = XOR_VX_VY
		case 0x4:
			ins.kind = ADD_VX_VY
		case 0x5:
			ins.kind = SUB_VX_VY
		case 0x6:
			ins.kind = SHR_VX_VY
		case 0x7:
			ins.kind = SUBN_VX_VY
		case 0xE:
			ins.kind = SHL_VX_VY
		}
	case 0x9:
		if ins.get_n() == 0x0 {
			ins.kind = SNE_VX_VY
		}
	case 0xA:
		ins.kind = LD_I_NNN
	case 0xB:
		ins.kind = JMP_V0_NNN
	case 0xC:
		ins.kind = RND_VX_NN
	case 0xD:
		ins.kind = DRW_VX_VY_N
	case 0xE:
		switch ins.get_nn() {
		case 0x9E:
			ins.kind = SKP_VX
		case 0xA1:
			ins.kind = SKNP_VX
		}
	case 0xF:
		switch ins.get_nn() {
		case 0x07:
			ins.kind = LD_VX_DT
		case 0x0A:
			ins.kind = LD_VX_K
		case 0x15:
			ins.kind = LD_DT_VX
		case 0x18:
			ins.kind = LD_ST_VX
		case 0x1E:
			ins.kind = ADD_I_VX
		case 0x29:
			ins.kind = LD_F_VX
		case 0x33:
			ins.kind = LD_B_VX
		case 0x55:
			ins.kind = LD_I_VX
		case 0x65:
			ins.kind = LD_VX_I
		}
	}
	return ins
}

func (chip *Chip) execute(ins Instruction) error {
	x := ins.get_x()
	y := ins.get_y()
	n := ins.get_n()
	nn := ins.get_nn()
	nnn := ins.get_nnn()
	switch ins.kind {
	case CLS: // 00E0
		chip.display = [SCREEN_HEIGHT][SCREEN_WIDTH]bool{}
		chip.redraw = true
	case RET: // 00EE
		chip.pc = uint16(chip.memory[chip.sp]) << 8
		chip.sp++
		chip.pc += uint16(chip.memory[chip.sp])
		chip.sp++
	case JMP: // 1NNN
		jmp_addr := ins.get_nnn()
		if O_HLT_ON_LOOP && chip.pc == jmp_addr+2 {
			chip.Draw()
			panic("Halt (infinite loop detected)")
		}
		chip.pc = jmp_addr
	case CALL_NNN: // 2NNN
		chip.sp--
		chip.memory[chip.sp] = uint8(chip.pc & 0x00FF)
		chip.sp--
		chip.memory[chip.sp] = uint8((chip.pc & 0xFF00) >> 8)
		chip.pc = nnn
	case SE_VX_NN: // 3NNN
		if chip.V[x] == nn {
			chip.pc += 2
		}
	case SNE_VX_NN: // 4XNN
		if chip.V[x] != nn {
			chip.pc += 2
		}
	case SE_VX_VY: // 5XY0
		if chip.V[x] == chip.V[y] {
			chip.pc += 2
		}
	case LD_VX_NN: // 6XNN
		chip.V[x] = nn
	case ADD_VX_NN: // 7XNN
		chip.V[x] += nn
	case LD_VX_VY: // 8XY0
		chip.V[x] = chip.V[y]
	case OR_VX_VY: // 8XY1
		chip.V[x] |= chip.V[y]
	case AND_VX_VY: // 8XY2
		chip.V[x] &= chip.V[y]
	case XOR_VX_VY: // 8XY3
		chip.V[x] ^= chip.V[y]
	case ADD_VX_VY: // 8XY4
		if chip.V[x]+chip.V[y] > 0xFF {
			chip.V[0xF] = 1
		} else {
			chip.V[0xF] = 0
		}
		chip.V[x] += chip.V[y]
	case SUB_VX_VY: // 8XY5
		if chip.V[x] > chip.V[y] {
			chip.V[0xF] = 1
		} else {
			chip.V[0xF] = 1
		}
		chip.V[x] -= chip.V[y]
	case SHR_VX_VY: // 8XY6
		chip.V[0xF] = chip.V[x] & 0x01
		chip.V[x] = chip.V[x] >> 1
	case SUBN_VX_VY: // 8XY7
		if chip.V[y] > chip.V[x] {
			chip.V[0xF] = 1
		} else {
			chip.V[0xF] = 0
		}
		chip.V[x] -= -chip.V[y] // magic operator
	case SHL_VX_VY: // 8XYE
		chip.V[0xF] = chip.V[x] & 0x80
		chip.V[x] = chip.V[x] << 1
	case SNE_VX_VY: // 9XY0
		if chip.V[x] != chip.V[y] {
			chip.pc += 2
		}
	case LD_I_NNN: // ANNN
		chip.I = nnn
	case JMP_V0_NNN: // BNNN
		chip.pc = uint16(chip.V[0]) + nnn
	case RND_VX_NN: // CXNN
		rand.Seed(time.Now().UnixNano())
		chip.V[x] = uint8(rand.Intn(256)) & nn
	case DRW_VX_VY_N: // DXYN
		xcoord := chip.V[x]
		ycoord := chip.V[y]
		for row := uint8(0); row < n; row++ {
			bits := chip.memory[chip.I+uint16(row)]
			cy := (ycoord + row) % SCREEN_HEIGHT
			for col := uint8(0); col < 8; col++ {
				cx := (xcoord + col) % SCREEN_WIDTH
				curr_col := chip.display[cy][cx]
				colb := bits & (0x01 << (7 - col))
				if colb > 0 {
					if curr_col {
						chip.display[cy][cx] = false
						chip.V[0xF] = 1
					} else {
						chip.display[cy][cx] = true
					}
				}
				if cx == SCREEN_WIDTH-1 {
					break
				}
			}
			if cy == SCREEN_HEIGHT-1 {
				break
			}
		}
		chip.redraw = true
	case SKP_VX: // EX9E
		break
	case SKNP_VX: // EXA1
		break
	case LD_VX_DT: // FX07
		chip.V[x] = chip.dt
	case LD_VX_K: // FX0A
		val, _ := strconv.ParseUint(Input(""), 16, 8)
		chip.V[x] = uint8(val)
	case LD_DT_VX: // FX15
		chip.dt = chip.V[x]
	case LD_ST_VX: // FX18
		chip.st = chip.V[x]
	case ADD_I_VX: // FX1E
		chip.I += uint16(chip.V[x])
	case LD_F_VX: // FX29
		chip.I = uint16(chip.V[x]) * 0x05
	case LD_B_VX: // FX33
		h := chip.V[x] / 100
		t := (chip.V[x] - h*100) / 10
		o := chip.V[x] - h*100 - t*10

		chip.memory[chip.I] = h
		chip.memory[chip.I+1] = t
		chip.memory[chip.I+2] = o
	case LD_I_VX: // FX55
		for reg := uint16(0); reg <= uint16(x); reg++ {
			chip.memory[chip.I+reg] = chip.V[reg]
		}
	case LD_VX_I: // FX65
		for reg := uint16(0); reg <= uint16(x); reg++ {
			chip.V[reg] = chip.memory[chip.I+reg]
		}
	default:
		return errors.New("Invalid instruction: " + fmt.Sprintf("%x", ins.data))
	}
	return nil
}

/*
	func (chip *Chip)decode_and_execute(data uint16) {
	    switch data >> 12 {
	        case 0x1: // jmp addr
	            jmp_addr := data & 0x0FFF
	            if O_HLT_ON_LOOP && chip.pc == jmp_addr+2{
	                chip.Draw()
	                panic("Halt (infinite loop detected)")
	            }
	            chip.pc = jmp_addr
	            show_disasm("jp " + strconv.FormatUint(uint64(jmp_addr), 16))
	        case 0x2: // call nnn
				panic(0x2)
	        case 0x3: // se vx nn
				x := uint8((data & 0x0F00) >> 8)
				nn := uint8(data & 0x00FF)
				if chip.V[x] == nn {
					chip.pc += 2
				}
				show_disasm("se v" + strconv.FormatUint(uint64(x), 16) + " " + strconv.FormatUint(uint64(nn), 16))
	        case 0x4: // sne vx nn
				x := uint8((data & 0x0F00) >> 8)
				nn := uint8(data & 0x00FF)
				if chip.V[x] != nn {
					chip.pc += 2
				}
				show_disasm("sne v" + strconv.FormatUint(uint64(x), 16) + " " + strconv.FormatUint(uint64(nn), 16))
	        case 0x5: // se vx vy
				x := uint8((data & 0x0F00) >> 8)
				y := uint8((data & 0x00F0) >> 4)
				if chip.V[x] == chip.V[y] {
					chip.pc += 2
				}
				show_disasm("se v" + strconv.FormatUint(uint64(x), 16) + " v" + strconv.FormatUint(uint64(y), 16))
	        case 0x6: // ld vx byte
	            x := uint8((data >> 8) & 0x0F)
	            chip.V[x] = uint8(data & 0x00FF)
	            show_disasm("ld v" + strconv.FormatUint(uint64(x), 16) + " 0x" + strconv.FormatUint(uint64(data & 0x00FF), 16))
	        case 0x7: // add vx byte
	            x := (data & 0x0F00) >> 8
	            chip.V[x] += uint8(data & 0x00FF)
	            show_disasm("add v" + strconv.FormatUint(uint64(x), 16) + " " + strconv.FormatUint(uint64(data & 0x00FF), 16))
	        case 0x8:
	            panic(0x8)
	        case 0x9: // sne vx vy
				x := uint8((data & 0x0F00) >> 8)
				y := uint8((data & 0x00F0) >> 4)
				if chip.V[x] != chip.V[y] {
					chip.pc += 2
				}
				show_disasm("se v" + strconv.FormatUint(uint64(x), 16) + " v" + strconv.FormatUint(uint64(y), 16))
	        case 0xa: // ld i addr
	            chip.I = data & 0x0FFF
	            show_disasm("ld I 0x" + strconv.FormatUint(uint64(data & 0x0FFF), 16))
	        case 0xc:
	            panic(0xc)
	        case 0xd: // drw vx vy nibble
	            x := uint8((data & 0x0F00) >> 8)
	            xcoord := chip.V[x]
	            y := uint8((data & 0x00F0) >> 4)
	            ycoord := chip.V[y]
	            n := uint8(data & 0x000F)
	            for row := uint8(0); row < n; row++ {
	                bits := chip.memory[chip.I + uint16(row)]
	                cy := (ycoord + row) % SCREEN_HEIGHT
	                for col := uint8(0); col < 8; col++{
	                    cx := (xcoord + col) % SCREEN_WIDTH
	                    curr_col := chip.display[cy][cx]
	                    colb := bits & (0x01 << (7 - col))
	                    if colb > 0{
	                        if curr_col{
	                            chip.display[cy][cx] = false
	                            chip.V[0xF] = 1
	                        } else {
	                            chip.display[cy][cx] = true
	                        }
	                    }
	                    if cx == SCREEN_WIDTH - 1{
	                        break
	                    }
	                }
	                if cy == SCREEN_HEIGHT - 1{
	                    break
	                }
	            }
	            chip.redraw = true
	            show_disasm("drw v"+strconv.FormatUint(uint64(x), 16)+"("+strconv.FormatUint(uint64(xcoord), 16)+") v"+strconv.FormatUint(uint64(y), 16)+"("+strconv.FormatUint(uint64(ycoord), 16)+") "+strconv.FormatUint(uint64(n), 16))
	        case 0xe:
	            panic(0xe)
	        default:
	            if data == 0x00E0{
	                chip.display = [SCREEN_HEIGHT][SCREEN_WIDTH]bool{}
	                chip.redraw = true
	                show_disasm("cls")
	            } else if data == 0x00EE{
	                show_disasm("ret")
	            }
	    }
	}
*/
func (chip *Chip) step(breakpoints []uint16) bool {
	data := chip.fetch()
	ins := chip.decode(data)
	//fmt.Printf("%v\n", ins.String())
	err := chip.execute(ins)
	if err != nil {
		chip.dump()
		panic(err)
	}
	for _, x := range breakpoints {
		if x == chip.pc {
			return true
		}
	}
	return false
}

func (chip *Chip) Run(breakpoints []uint16) {
	done := false
	for !done {
		done = chip.step(breakpoints)
	}
}

func Input(msg string) string {
	fmt.Print(msg)
	scanner := bufio.NewScanner(os.Stdin)
	var line string
	if scanner.Scan() {
		line = scanner.Text()
	}
	return line
}

/*
func main() {
	rom := "roms/test_opcode.ch8"
	for true {
		fmt.Println("loading " + rom)
		chip := NewChip(rom)
		done := false
		var breakpoints []uint16
		for !done{
			cmd := strings.Split(Input("(r)eset, (s)tep, (c)ontinue, (d)ebug, dra(w), (b)reak [addr] > "), " ")
			switch cmd[0] {
			case "r":
				done = true
			case "c":
				chip.Run(breakpoints)
			case "d":
				chip.dump()
			case "w":
				chip.Draw()
			case "b":
				if len(cmd) > 1 {
					addr, _ := strconv.ParseUint(cmd[1], 16, 16)
					breakpoints = append(breakpoints, uint16(addr))
				} else {
					fmt.Println("specify an address")
				}
			default:
				chip.step(breakpoints)
				chip.dump()
			}
		}
	}
}
*/
