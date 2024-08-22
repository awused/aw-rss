fn main() {
    println!("cargo:rerun-if-changed=src/database/migrations");
    println!("cargo:rerun-if-changed=dist");
}
