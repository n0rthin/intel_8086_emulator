import fs from "node:fs";

setImmediate(main);

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
register_op(OPS, "1100011x", mov_immediate_to_reg_or_mem);
register_op(OPS, "1010000x", mov_mem_to_accum);
register_op(OPS, "1010001x", mov_accum_to_mem);
register_op(OPS, "000000xx", add_reg_to_reg);
register_op(OPS, "100000xx", arithm_immediate_to_reg_or_mem);
register_op(OPS, "0000010x", add_immediate_to_reg);
register_op(OPS, "001010xx", sub_reg_to_reg);
register_op(OPS, "100000xx", arithm_immediate_to_reg_or_mem);
register_op(OPS, "0010110x", sub_immediate_to_reg);

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
  if (DEBUG) byte_reader.set_mark();
  while ((op = byte_reader.next()) !== null) {
    const handler = OPS.get(op);
    if (!handler) {
      console.log(asm);
      throw new Error(`Unkown operator ${op.toString(2)}`);
    };
    
    asm += "\n";
    let op_asm = handler(op, byte_reader);

    if (DEBUG) {
      op_asm = `(${byte_reader.get_bytes_since_mark_str()}) ${op_asm}`;
      byte_reader.set_mark();
    }

    asm += op_asm;
  }
  
  console.log(asm);
}

function read_reg_or_m(REG_OR_M: number, W: number, MOD: number, byte_reader: BytecodeReader): string {
  let reg_or_m: string;
  switch (MOD) {
    case 0b11:
      reg_or_m = reg_to_label(REG_OR_M, W)
      break;
    case 0b00:
      if (REG_OR_M == 0b110) {
        // direct address
        reg_or_m = `[${byte_reader.read_number(W ? 2 : 1)}]`;
      } else {
        reg_or_m = `[${reg_to_addr_calc(REG_OR_M).join(" + ").toLowerCase()}]`;
      }
      break;
    case 0b01:
    case 0b10:
      const disp = byte_reader.read_number(MOD);
      const address_formula: Array<string | number> = [...reg_to_addr_calc(REG_OR_M)];
      if (disp > 0) {
        address_formula.push(disp);
      } 
      reg_or_m = `[${address_formula.join(" + ").toLowerCase()}`;
      if (disp < 0) {
        reg_or_m += ` - ${Math.abs(disp)}]`;
      } else {
        reg_or_m += "]";
      }
      break;
    default:
      reg_or_m = reg_to_label(REG_OR_M, W)
      break;
  } 

  return reg_or_m;
}

function mov_reg_to_reg(op: number, byte_reader: BytecodeReader): string {
  return reg_to_reg("mov", op, byte_reader);
}

function add_reg_to_reg(op: number, byte_reader: BytecodeReader): string {
  return reg_to_reg("add", op, byte_reader);
}

function sub_reg_to_reg(op: number, byte_reader: BytecodeReader): string {
  return reg_to_reg("sub", op, byte_reader);
}

function reg_to_reg(op_label: string, op: number, byte_reader: BytecodeReader): string {
  const D_MASK = 0b00000010;
  const W_MASK = 0b00000001;

  const D = match_mask(D_MASK, op);
  const W = match_mask(W_MASK, op);
  const opeands =  byte_reader.next();
  const MOD = opeands >> 6;
  const REG = (opeands >> 3) & 0b111;
  const REG_OR_M = opeands & 0b111;

  let asm = `${op_label} `;

  const reg = reg_to_label(REG, W);
  const reg_or_m: string = read_reg_or_m(REG_OR_M, W, MOD, byte_reader);

  let src: string, dest: string;
  if (D) {
    dest = reg;
    src = reg_or_m;
  } else {
    dest = reg_or_m;
    src = reg;
  }

  asm += `${dest}, ${src}`;

  return asm;
}

function mov_immediate_to_reg(op: number, byte_reader: BytecodeReader): string {
  return immediate_to_reg("mov", op, byte_reader);
}

function add_immediate_to_reg(op: number, byte_reader: BytecodeReader): string {
  return immediate_to_reg("add", op, byte_reader);
}

function sub_immediate_to_reg(op: number, byte_reader: BytecodeReader): string {
  return immediate_to_reg("sub", op, byte_reader);
}

function immediate_to_reg(op_label: string, op: number, byte_reader: BytecodeReader): string {
  const W_MASK = op_label === "mov" ? 0b00001000 : 0b00000001;
  const W = match_mask(W_MASK, op);
  const REG = op_label === "mov" ? op & 0b111 : 0b000;
  const DATA = byte_reader.read_number(W ? 2 : 1);
    
  let asm = `${op_label} `;
  asm += reg_to_label(REG, W) + ", " + DATA

  return asm;
}

function mov_immediate_to_reg_or_mem(op: number, byte_reader: BytecodeReader): string {
  return immediate_to_reg_or_mem("mov", op, byte_reader);
}

function arithm_immediate_to_reg_or_mem(op: number, byte_reader: BytecodeReader): string {
  const operator = op >> 3 & 0b111;
  switch (operator) {
    case 0b000:
      return immediate_to_reg_or_mem("add", op, byte_reader);
    case 0b101:
      return immediate_to_reg_or_mem("sub", op, byte_reader);
    default:
      throw new Error(`Uknown operator ${op}`)
  }
}

function immediate_to_reg_or_mem(op_label: string, op: number, byte_reader: BytecodeReader): string {
  const W_MASK = 0b00000001;
  const S_MASK = 0b00000010;

  let W = match_mask(W_MASK, op);
  const S = match_mask(S_MASK, op);
  const opeands =  byte_reader.next();
  const MOD = opeands >> 6;
  const REG_OR_M = opeands & 0b111;

  let asm = `${op_label} `;

  const reg_or_m: string = read_reg_or_m(REG_OR_M, W, MOD, byte_reader);
  W = Number(W && !S) as 0 | 1;
  const data = byte_reader.read_number(W ? 2 : 1);

  asm += `${reg_or_m}, ${W ? "word" : "byte"} ${data}`;

  return asm;
}

function mov_mem_to_accum(op: number, byte_reader: BytecodeReader): string {
  const W_MASK = 0b00000001;

  const W = match_mask(W_MASK, op);
  const addr =  byte_reader.read_number(2);

  const asm = `mov ${reg_to_label(0b000, W)}, [${addr}]`;
  return asm;
}

function mov_accum_to_mem(op: number, byte_reader: BytecodeReader): string {
  const W_MASK = 0b00000001;

  const W = match_mask(W_MASK, op);
  const addr =  byte_reader.read_number(2);

  const asm = `mov [${addr}], ${reg_to_label(0b00, W)}`;
  return asm;
}

function match_mask(mask: number, data: number): 1 | 0 {
  return Number((data & mask) === mask) as 1 | 0; 
}