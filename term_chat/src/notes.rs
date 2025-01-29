use crate::tui::{clear, input};

fn add_note<'a>(inp: &'a str, notes: &mut Vec<String>) {
    println!("Adding note to local notes: {}", inp);
    notes.push(inp.to_string());
}

fn makereq(command: &str, text: &str) {
    // TODO: Implement making HTTP requests
    println!("Making request: {} {}", command, text);
}

fn get_notes(notes: &Vec<String>) {
    println!("Getting notes from local notes:");
    for note in notes {
        println!("{}", note);
    }
}

fn register() {
    print!("Input your username:");
    let username = match input() {
        Ok(input) => input,
        Err(e) => {
            eprintln!("Error reading input: {}", e);
            return;
        }
    };
    print!("Input your password:");
    let password = match input() {
        Ok(input) => input,
        Err(e) => {
            eprintln!("Error reading input: {}", e);
            return;
        }
    };
    println!("Creating new account with username: {}, password: {}", username, password);
    // TODO: Add registration logic here
}

pub fn init() {
    let mut local_notes: Vec<String> = vec![];
    loop {
        let inp = match input() {
            Ok(input) => input,
            Err(e) => {
                eprintln!("Error reading input: {}", e);
                continue;
            }
        };

        let trimmed_input = inp.trim();

        match trimmed_input {
            "exit" => return,
            "clear" => clear(),
            "register" => register(),
            "get" => get_notes(&local_notes),
            _ => add_note(trimmed_input, &mut local_notes),
        }
    }
}
