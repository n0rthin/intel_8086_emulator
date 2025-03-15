import fs from "node:fs";

main();

const DEBUG = process.env.DEBUG;
const REG_FIELD_ENCODING = new Map([
  [0b000, new Map([[0, "AL"], [1, "AX"]])],
  [0b001, new Map([[0, "CL"], [1, "CX"]])],
  [0b010, new Map([[0, "DL"], [1, "DX"]])],
  [0b011, new Map([[0, "BL"], [1, "BX"]])],
  [0b100, new Map([[0, "AH"], [1, "SP"]])],
  [0b101, new Map([[0, "CH"], [1, "BP"]])],
  [0b110, new Map([[0, "DH"], [1, "SI"]])],
  [0b111, new Map([[0, "BH"], [1, "DI"]])],
]);
function reg_to_label(reg: number, W: number): string | never {
  const reg_map = REG_FIELD_ENCODING.get(reg);
  if (!reg_map) throw new Error(`No encoding for reg 0b${reg.toString(2).padStart(3, "0")} was found`);
  const label = reg_map.get(W);
  if (!label) throw new Error(`No encoding for reg 0b${reg.toString(2).padStart(3, "0")} and W ${W} was found`); 
  return label.toLowerCase();
}
const EFFECTIVE_ADDR_CALC_ENCODING = new Map([
  [0b000, ["BX", "SI"]],
  [0b001, ["BX", "DI"]],
  [0b010, ["BP", "SI"]],
  [0b011, ["BP", "DI"]],
  [0b100, ["SI"]],
  [0b101, ["DI"]],
  [0b110, ["BP"]],
  [0b111, ["BX"]],
]);
function reg_to_addr_calc(reg: number): readonly string[] | never {
  const addr_calc = EFFECTIVE_ADDR_CALC_ENCODING.get(reg);
  if (!addr_calc) throw new Error(`No effective address calculation encoding for reg 0b${reg.toString(2).padStart(3, "0")} was found`);
  return Object.freeze(addr_calc);
}

type OpHandler = (op: number, byte_reader: BytecodeReader) => string;
const OPS = new Map<number, OpHandler>();
register_op(OPS, "100010xx", mov_reg_to_reg);
register_op(OPS, "1011xxxx", mov_immediate_to_reg);

function register_op(ops_map: Map<number, OpHandler>, op: string, handler: OpHandler): void {
  const variable_parts_count = op.split("x").length - 1;
  const variants = 2 ** variable_parts_count;
  for (let i = 0; i < variants; i++) {
    let variant = op;
    i
      .toString(2)
      .padStart(variable_parts_count, "0")
      .split('')
      .forEach(part => variant = variant.replace("x", part));
    ops_map.set(parseInt(variant, 2), handler);
  }
}

class BytecodeReader {
  private curr: number;
  private mark: number | null;
  
  constructor(private readonly bytecode: Buffer) { 
    this.curr = 0;
    this.mark = null;
  }

  public get pos(): number {
    return this.curr;
  }

  public next(): number {
    return this.bytecode[this.curr++];
  }


  public read_number(bytes: number = 1): number {
    let n = this.next();
    for (let i = 1; i < bytes; i++) {
      const next_byte = this.next();
      n = (next_byte << 8 * i) | n;
    }
    return decode_number(n, bytes * 8);
  }

  public tail(n: number): Buffer {
    return this.bytecode.subarray(this.curr - n, this.curr);
  }

  public set_mark(): void {
    this.mark = this.curr;
  }

  public get_bytes_since_mark_str(): string | never {
    if (this.mark === null) throw new Error("Can't get bytes since mark because mark is not set");
    return [...this.tail(this.curr - this.mark)].map(b => b.toString(2).padStart(8, "0")).join(" ");
  }
}

function decode_number(n: number, bits: number): number {
  if (n >> (bits - 1) === 1) n -= 2 ** bits;
  return n;
}

function main() {
  const filepath = process.argv[2];
  const bytecode = fs.readFileSync(filepath);
  const byte_reader = new BytecodeReader(bytecode);
  
  let asm = "bits 16\n";
  let op: number;
  byte_reader.set_mark();
  while (!!(op = byte_reader.next())) {
    const handler = OPS.get(op);
    if (!handler) {
      console.log(asm)
      throw new Error(`Unkown operator ${op.toString(2)}`);
    };
    
    asm += "\n";
    asm += handler(op, byte_reader);
    byte_reader.set_mark();
  }
  
  console.log(asm);
}

function mov_reg_to_reg(op: number, byte_reader: BytecodeReader): string {
  const D_MASK = 0b00000010;
  const W_MASK = 0b00000001;

  const D = match_mask(D_MASK, op);
  const W = match_mask(W_MASK, op);
  const operands =  byte_reader.next();
  const MOD = operands >> 6;
  const REG = (operands >> 3) & 0b111;
  const REG_OR_M = operands & 0b111;

  let asm = `mov `;

  const reg = reg_to_label(REG, W);
  let reg_or_m: string;
  switch (MOD) {
    case 0b11:
      reg_or_m = reg_to_label(REG_OR_M, W)
      break;
    case 0b00:
      reg_or_m = `[${reg_to_addr_calc(REG_OR_M).join(" + ").toLowerCase()}]`;
      break;
    case 0b01:
    case 0b10:
      const disp = byte_reader.read_number(MOD);
      const address_formula: Array<string | number> = [...reg_to_addr_calc(REG_OR_M)];
      if (disp) address_formula.push(disp)
      reg_or_m = `[${address_formula.join(" + ").toLowerCase()}]`;
      break;
    default:
      reg_or_m = reg_to_label(REG_OR_M, W)
      break;
  }

  let src: string, dest: string;
  if (D) {
    dest = reg;
    src = reg_or_m;
  } else {
    dest = reg_or_m;
    src = reg;
  }

  asm += `${dest}, ${src}`;

  if (DEBUG) {
    asm = `(${byte_reader.get_bytes_since_mark_str()}) ${asm}`;
  }

  return asm;
}

function mov_immediate_to_reg(op: number, byte_reader: BytecodeReader): string {
  const W_MASK = 0b00001000;
  const W = match_mask(W_MASK, op);
  const REG = op & 0b111;
  const DATA = byte_reader.read_number(W ? 2 : 1);
    
  let asm = `mov `;
  asm += reg_to_label(REG, W) + ", " + DATA

  if (DEBUG) {
    asm = `(${byte_reader.get_bytes_since_mark_str()}) ${asm}`;
  }

  return asm;
}

function match_mask(mask: number, data: number): 1 | 0 {
  return Number((data & mask) === mask) as 1 | 0; 
}