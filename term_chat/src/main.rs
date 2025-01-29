mod notes;
mod tui;
use clap::{App, Arg};
use crossterm::event::{KeyCode, KeyEvent,self};
use tui::clear;
fn main() {
    let args = App::new("notes")
        .version("0.01")
        .about("Make notes, chat with ai")
        .arg(Arg::with_name("pattern")
            .help("Name of style")
            .takes_value(true)
            .required(false))
        .get_matches();
    let some =  args.value_of("pattern");
    let style = {
        if some == None {"notes"}
        else {
            some.unwrap().trim()
        }
    };
    clear();
    match style {
        "ai" => print!("I will do later"),
        "notes" => notes::init(),
        _ => notes::init()
    }
}
