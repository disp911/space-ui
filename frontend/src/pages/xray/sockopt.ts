// Checks of the socket options that Xray makes only when it dials or listens.

const number = /^\d+$/

/** A custom option without a numeric opt: Xray refuses the connection ("No opt!"). */
export const optWrong = (c: Record<string, any>) => !number.test(String(c.opt ?? '').trim())

/**
 * Custom options Xray cannot set: no numeric opt, a level that is no number,
 * a type other than int or str, or an int value that is no number.
 */
export function customSockoptWrong(so: Record<string, any> | undefined): boolean {
  const list = Array.isArray(so?.customSockopt) ? so.customSockopt : []
  return list.some((c: Record<string, any>) => {
    const level = String(c.level ?? '').trim()
    const type = String(c.type ?? '')
    return optWrong(c) || (!!level && !number.test(level)) || !['int', 'str'].includes(type) || (type === 'int' && !/^-?\d+$/.test(String(c.value ?? '').trim()))
  })
}
