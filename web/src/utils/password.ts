import {sha256} from "js-sha256"

/**
 * sha256加密密码
 * @param password
 */
export function encodePassword(password: string): string {
    return sha256(password)
}
