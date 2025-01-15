const socket = new WebSocket(`ws://${location.host}/ws`); // Замените на реальный URL вебсокета
socket.addEventListener('open', function (event) {
    console.log('Соединение с вебсокетом открыто!');
});
let LASTID = -1
let FIRSTID = null
const ALLNOTES = []
socket.addEventListener('message', async function (event) {
    const notes = await makeRequest({
        action: "get_notes",
        log: "Takesomebigger",
        someid: LASTID,
    })
    console.log("notes", notes)
    SearchMimMaxIdNotes(notes);
    displayNotes()
});

onload = () => {
    if (location.hash === "") {
        location = "#login"
    }
    else {
        onpopstate()
    }
};

let user = "";
let password = "";
onpopstate = async () => {
    if (location.hash === "#login") {
        drawLogin(document.getElementById('something'))
    }
    if (location.hash === "#notes") {
        drawNotes(document.getElementById('something'))
        const notes = await makeRequest({
            action: "get_notes",
            log: "Takefirst",
            howmuch: 3,

        })

        SearchMimMaxIdNotes(notes);
        displayNotes();
    }
}


function drawLogin(element) {
    element.innerHTML = `
    <body>
        <div class="login-form">
            <h2>Вход</h2>
            <div class="input-form">
                <input id="username" placeholder="Введите ваш никнейм..." required></input>
                <input id="password" placeholder="Введите ваш пароль..." required></input>
                <button id="textbutton">Войти</button>
            </div>
            <div id="COUNTER" style="text-align: center; margin-bottom: 15px;"></div>
            <div id="ERROR" style="color: red; text-align: center;"></div> <!-- Элемент для отображения ошибок -->
            
        </div>
        
    </body>
    `
    COUNTER = document.getElementById('COUNTER');
    ERROR = document.getElementById('ERROR');

    document.getElementById('textbutton').onclick = async () => {
        user = document.getElementById('username').value;
        password = document.getElementById('password').value;
        responce = await makeRequest({
            action: "login",
            user: user,
            pass: password,
        })
        location = "#notes"
    }

}

function drawNotes(element) {
    element.innerHTML = `
    <body>
    <div class="note-form">
        <button id = "Back">Log in or rename</button>
        <h2>Заметки</h2>
        <button id = "NewNotes">Load older notes</button>
        <div id="COUNTER""></div> <!-- Счетчик выше поля ввода -->
            <h2>Сохранить заметку</h2>
            <div class="input-form">
                <textarea id="note" placeholder="Введите вашу заметку..." required></textarea>
                <button id="textbutton">Сохранить</button>
            </div>
            <div id="ERROR" style="color: red; text-align: center;"></div> <!-- Элемент для отображения ошибок -->
        </div>
    </body>
    `
    //element.querySelector("#button").onclick = () => {
    //    location = "#screen1"
    //}
    COUNTER = document.getElementById('COUNTER');
    ERROR = document.getElementById('ERROR');

    element.querySelector("#Back").onclick = () => {
        location = "#login"
    }
    document.getElementById('textbutton').onclick = async () => {
        const text = document.getElementById('note');

        await makeRequest({
            action: "add_note",
            message: user + "," + text.value,
        })
    }

    let lock = false;
    const scrollBoundary = 10;
    const newNotesButton = document.getElementById('NewNotes');
    newNotesButton.onclick = async () => {
        //const text = document.getElementById('note');
        if (lock) { return; }
        try {
            lock = true;
            newNotesButton.disabled = true;

            const notes = await makeRequest({
                action: "get_notes",
                log: "Takesomelower",
                howmuch: 3,
                someid: FIRSTID,
            })
            SearchMimMaxIdNotes(notes);
            displayNotes(scrollBoundary)
        } finally { lock = false; newNotesButton.disabled = false; }
    }
    COUNTER.onscroll = () => {
        console.log("SCROLL", COUNTER.scrollTop)
        if (COUNTER.scrollTop < scrollBoundary) {
            if (newNotesButton){
                newNotesButton.click()
            }
        }
        
    };

}

function sanitize(html) {
    const tempDiv = document.createElement('div');
    tempDiv.textContent = html; // Set text content to escape HTML
    return tempDiv.innerHTML; // Get the escaped HTML
}

function displayNotes(scrollBoundary) {
    console.log(ALLNOTES);

    COUNTER.innerHTML = "";
    ALLNOTES.forEach(note => {
        const noteElement = document.createElement('div'); // Создаем новый элемент для заметки
        noteElement.classList.add("myclassss");
        //console.log()
        noteElement.innerHTML = `<div class="note_header">${sanitize(note.username)}</div><div class="note_message">${sanitize(note.note)}</div>`; // Устанавливаем текст заметки
        COUNTER.appendChild(noteElement); // Добавляем элемент в COUNTER
    });

    if (scrollBoundary) { COUNTER.scrollTop = scrollBoundary; }
}

async function makeRequest(request) {
    try {

        const response = await fetch("", {
            method: "POST",
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(request)
        });

        if (!response.ok) {
            ERROR.innerHTML = "ОШИБКА ДОСТУПА К СЕРВЕРУ";
            return;
        }
        const result = await response.json();
        console.log("result", result)
        return result

    } catch (error) {
        console.error("Ошибка при отправке запроса:", error, "  Запрос", request);
        // TODO: возможно, нужно бросить дальше
    }
}


function SearchMimMaxIdNotes(notes) {
    for (const note of notes) {
        ALLNOTES.push(note);
        if (FIRSTID === null || FIRSTID > note.id) {
            FIRSTID = note.id;
        }
        if (LASTID === null || LASTID < note.id) {
            LASTID = note.id;
        }
    }

    ALLNOTES.sort((a, b) => a.id - b.id);
}




async function doLikeButtom(scroll_pos) {
    
}