use std::process::Command;
use std::io::{self, Write};

pub fn input() -> Result<String, io::Error> {
    print!(" > ");
    io::stdout().flush()?;
    let mut text = String::new();
    io::stdin().read_line(&mut text)?;
    Ok(text.trim().to_string())
}

pub fn clear() {
    let result = if cfg!(target_os = "windows") {
        Command::new("cmd").args(["/c", "cls"]).status()
    } else {
        Command::new("tput").arg("reset").status()
    };

    if let Err(_) = result {
        print!("{esc}c", esc = 27 as char);
        io::stdout().flush().unwrap();
    }

    
}
