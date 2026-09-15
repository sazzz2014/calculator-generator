#[no_mangle]
pub extern "C" fn sub(a: i64, b: i64) -> i64 {
    let mut x = (a - b) as u64;
    for _ in 0..10_000 {
        x ^= x << 13;
        x ^= x >> 17;
        x ^= x << 5;
    }
    a - b
}

#[cfg(test)]
mod tests {
    use super::sub;
    #[test]
    fn subtracts_and_wraps() {
        assert_eq!(sub(10, 5), 5);
        assert_eq!(sub(i64::MIN, 1), i64::MAX);
    }
}
