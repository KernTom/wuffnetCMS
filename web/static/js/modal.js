// Dynamisch Form-Felder erzeugen und Werte füllen (Neuer Eintrag / Bearbeiten)
async function loadFormFields(tableSchema, tableName, record = {}) {
    console.log("Lade Formularfelder für Tabelle", tableName);
    const formFields = document.getElementById("form-fields");
    formFields.innerHTML = "";  // Reset des Formulars

    try {
        // Fetch Spalteninformationen
        const response = await fetch(`/api/table-fields?schema=${tableSchema}&table=${tableName}`);
        if (!response.ok) throw new Error(`Fehler beim Abrufen der Felder: ${response.statusText}`);
        const columns = await response.json();
        console.log("Spalteninformationen abgerufen:", columns);

        columns.forEach(column => {
            const fieldWrapper = document.createElement("div");
            fieldWrapper.className = "form-field-wrapper"; // Zweispaltige Anordnung

            const label = document.createElement("label");
            label.textContent = column.name;
            label.className = "form-label";
            label.setAttribute("for", column.name);

            let input;
            switch (column.type) {
                case "integer":
                    input = document.createElement("input");
                    input.type = "number";
                    input.value = record[column.name] || "";
                    input.name = column.name;
                    if (column.readonly) input.setAttribute("readonly", true);
                    fieldWrapper.appendChild(label);
                    fieldWrapper.appendChild(input);
                    formFields.appendChild(fieldWrapper);
                    break;

                    case "select":
                        input = document.createElement("select");
                        input.name = column.name;
                        input.id = column.name;
                        input.name = column.name;
                        if (column.readonly) input.setAttribute("readonly", true);
                        // Standardoption hinzufügen
                        const defaultOption = document.createElement("option");
                        defaultOption.text = "-- Bitte wählen --";
                        defaultOption.value = "";
                        input.appendChild(defaultOption);
    
                        // Füge die übergebenen Optionen mit Labels hinzu
                        column.options.forEach(option => {
                            const opt = document.createElement("option");
                            opt.value = option.value;
                            opt.text = option.label; // Label statt Wert anzeigen
                            if (record[column.name] == option.value) opt.selected = true;
                            input.appendChild(opt);
                        });
    
                        fieldWrapper.appendChild(label);
                        fieldWrapper.appendChild(input);
                        formFields.appendChild(fieldWrapper);
    
                        M.FormSelect.init(input); // Materialize Select initialisieren
                        break;
    

                case "float":
                    input = document.createElement("input");
                    input.type = "text";
                    input.name = column.name;
                    input.setAttribute("pattern", "^[0-9]+(,[0-9]{1,2})?$"); // Erlaubt 123 und 123,45
                    input.setAttribute("title", "Bitte eine gültige Zahl im Format 123,45 eingeben");
                    input.value = record[column.name] || "";
                    if (column.readonly) input.setAttribute("readonly", true);
                    // Verhindert Eingabe von Buchstaben und anderen unerwünschten Zeichen
                    input.addEventListener("input", (event) => {
                        // Überschreibe Eingabe mit dem, was dem Muster entspricht
                        input.value = input.value.replace(/[^0-9,]/g, ''); // Erlaubt nur Zahlen und Komma
                    });

                    fieldWrapper.appendChild(label);
                    fieldWrapper.appendChild(input);
                    formFields.appendChild(fieldWrapper);
                    break;

                case "boolean":
                    // Wrapper für den Switch
                    const switchWrapper = document.createElement("div");
                    switchWrapper.className = "switch";
                    switchWrapper.style.width = "100%"; 
                
                    // Label links für konsistentes Layout
                    label.style.fontWeight = "bold";
                    label.style.marginRight = "10px";
                    fieldWrapper.appendChild(label);
                
                    // Switch erstellen
                    input = document.createElement("input");
                    input.type = "checkbox";
                    input.id = column.name;
                    input.name = column.name;
                    // Sicherstellen, dass der Wert korrekt in einen Boolean konvertiert wird
                    input.checked = record[column.name] === true || record[column.name] === "true";
                
                    // Hebel für den Switch erstellen
                    const lever = document.createElement("span");
                    lever.className = "lever";
                
                    // Switch und Hebel in ein Label einschließen (Materialize-Layout)
                    const switchContainer = document.createElement("label");
                    switchContainer.appendChild(input);
                    switchContainer.appendChild(lever);
                
                    // Switch zum Wrapper hinzufügen und dem Layout einfügen
                    switchWrapper.appendChild(switchContainer);
                    fieldWrapper.appendChild(switchWrapper);
                
                    // Wrapper zum Formular hinzufügen
                    formFields.appendChild(fieldWrapper);
                    break;

                case "date":
                    input = document.createElement("input");
                    input.type = "text";
                    input.name = column.name;
                    input.className = "datepicker";
                    input.value = record[column.name] ? new Date(record[column.name]).toLocaleDateString("de-DE") : "";
                    fieldWrapper.appendChild(label);
                    fieldWrapper.appendChild(input);
                    formFields.appendChild(fieldWrapper);

                    setTimeout(() => {
                        M.Datepicker.init(input, {
                            autoClose: true,
                            format: "dd.mm.yyyy",
                            defaultDate: record[column.name] ? new Date(record[column.name]) : new Date(),
                            setDefaultDate: true,
                            firstDay: 1,
                            i18n: {
                                cancel: 'Abbrechen',
                                clear: 'Löschen',
                                done: 'Fertig',
                                months: ['Januar', 'Februar', 'März', 'April', 'Mai', 'Juni', 'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember'],
                                weekdays: ['Sonntag', 'Montag', 'Dienstag', 'Mittwoch', 'Donnerstag', 'Freitag', 'Samstag'],
                                weekdaysShort: ['So', 'Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa']
                            }
                        });
                    }, 0);
                    break;

                case "timestamp":
                    const timestampWrapper = document.createElement("div");
                    timestampWrapper.className = "timestamp-wrapper";
                    timestampWrapper.style.display = "flex";
                    timestampWrapper.style.gap = "10px";
                    timestampWrapper.style.width = "100%"; 

                    const dateInput = document.createElement("input");
                    dateInput.type = "text";
                    dateInput.className = "datepicker";
                    dateInput.name = `${column.name}_date`; 
                    dateInput.value = record[column.name] ? new Date(record[column.name]).toLocaleDateString("de-DE") : "";

                    const timeInput = document.createElement("input");
                    timeInput.type = "text";
                    timeInput.className = "timepicker";
                    timeInput.name = `${column.name}_time`;
                    timeInput.value = record[column.name] ? new Date(record[column.name]).toLocaleTimeString("de-DE", { hour: "2-digit", minute: "2-digit" }) : "";

                    timestampWrapper.append(dateInput, timeInput);
                    fieldWrapper.appendChild(label);
                    fieldWrapper.appendChild(timestampWrapper);
                    formFields.appendChild(fieldWrapper);

                    setTimeout(() => {
                        M.Datepicker.init(dateInput, {
                            autoClose: true,
                            format: "dd.mm.yyyy",
                            defaultDate: record[column.name] ? new Date(record[column.name]) : new Date(),
                            setDefaultDate: true,
                            firstDay: 1,
                            i18n: {
                                cancel: 'Abbrechen',
                                clear: 'Löschen',
                                done: 'Fertig',
                                months: ['Januar', 'Februar', 'März', 'April', 'Mai', 'Juni', 'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember'],
                                weekdays: ['Sonntag', 'Montag', 'Dienstag', 'Mittwoch', 'Donnerstag', 'Freitag', 'Samstag'],
                                weekdaysShort: ['So', 'Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa']
                            }
                        });

                        M.Timepicker.init(timeInput, {
                            twelveHour: false,
                            defaultTime: record[column.name] ? new Date(record[column.name]).toLocaleTimeString("de-DE", { hour: "2-digit", minute: "2-digit" }) : "now",
                            i18n: {
                                cancel: 'Abbrechen',
                                clear: 'Löschen',
                                done: 'Fertig'
                            }
                        });
                    }, 0);
                    break;

                    case "time":
                        input = document.createElement("input");
                        input.type = "text";
                        input.className = "timepicker";
                        input.name = column.name;
                        input.value = record[column.name] 
                            ? new Date(`1970-01-01T${record[column.name]}`).toLocaleTimeString("de-DE", { hour: "2-digit", minute: "2-digit" }) 
                            : "";
                    
                        fieldWrapper.appendChild(label);
                        fieldWrapper.appendChild(input);
                        formFields.appendChild(fieldWrapper);
                    
                        setTimeout(() => {
                            // Initialisiere den Timepicker mit richtigem Default-Wert und Formatierung
                            const timeValue = record[column.name] 
                                ? new Date(`1970-01-01T${record[column.name]}`).toLocaleTimeString("de-DE", { hour: "2-digit", minute: "2-digit" }) 
                                : "00:00";
                            
                            M.Timepicker.init(input, {
                                twelveHour: false,
                                defaultTime: timeValue,
                                i18n: {
                                    cancel: 'Abbrechen',
                                    clear: 'Löschen',
                                    done: 'Fertig'
                                }
                            });
                        }, 0);
                        break;

                    case "text":
                        // WYSIWYG-Editor erstellen
                        const editorContainer = document.createElement("div");
                        editorContainer.className = "wysiwyg-editor";
                        editorContainer.contentEditable = true;
                        editorContainer.style.minHeight = "6em";
                        editorContainer.innerHTML = record[column.name] || "";
                    
                        if (column.readonly) editorContainer.setAttribute("contenteditable", "false");
                    
                        // Verstecktes Eingabefeld erstellen, um den Inhalt zu speichern
                        input = document.createElement("input");
                        input.type = "hidden";
                        input.name = column.name; // Name für die spätere Verarbeitung setzen
                        input.value = record[column.name] || ""; // Wert setzen
                    
                        // Bei jeder Eingabe im WYSIWYG-Editor den Wert im versteckten Input aktualisieren
                        editorContainer.addEventListener("input", () => {
                            input.value = editorContainer.innerHTML; // Synchronisiere das versteckte Eingabefeld
                        });
                    
                        new MediumEditor(editorContainer, {
                            toolbar: {
                                allowMultiParagraphSelection: true,
                                buttons: ['bold', 'italic', 'underline', 'h2', 'h3', 'unorderedlist', 'orderedlist', 'quote', 'horizontalRule'],
                                static: true,
                                sticky: true,
                                align: 'left'
                            },
                            placeholder: { text: 'Schreiben Sie hier...' },
                            paste: { cleanPastedHTML: true }
                        });
                    
                        fieldWrapper.appendChild(label);
                        fieldWrapper.appendChild(editorContainer); // WYSIWYG-Editor hinzufügen
                        fieldWrapper.appendChild(input); // Verstecktes Eingabefeld hinzufügen
                        break;

                default:
                    input = document.createElement("input");
                    input.type = "text";
                    input.value = record[column.name] || "";
                    input.name = column.name;
                    if (column.readonly) input.setAttribute("readonly", true);
                    fieldWrapper.appendChild(label);
                    fieldWrapper.appendChild(input);
                    break;
            }
            formFields.appendChild(fieldWrapper);
        });

        M.updateTextFields();
    } catch (error) {
        console.error("Fehler bei der Feldinitialisierung:", error);
    }
}
// Öffnet das Modal und setzt ggf. Felder zurück
function openModal(record = null) {
    resetForm();

    const modal = document.getElementById("modal-container");
    const overlay = document.createElement("div");
    overlay.classList.add("modal-overlay");
    document.body.appendChild(overlay);

    modal.style.display = "block";
    overlay.style.display = "block";

    const modalTitle = document.getElementById("modal-title");
    if (record) {
        modalTitle.innerHTML = `Eintrag bearbeiten in <span class="tablecaption">${currentSchema}.${currentTable}</span>`;
        loadFormFields(currentSchema, currentTable, record); // Felder mit Daten füllen
    } else {
        modalTitle.innerHTML = `Neuen Eintrag erstellen in <span class="tablecaption">${currentSchema}.${currentTable}</span>`;
        loadFormFields(currentSchema, currentTable); // Leere Felder für neuen Eintrag
    }

    overlay.onclick = closeModal;
}

// Resettet das Formular, wenn keine Daten übergeben werden
function resetForm() {
    const formElements = document.querySelectorAll("#modal-container input, #modal-container select, #modal-container textarea");
    formElements.forEach(element => {
        if (element.type === "checkbox") {
            element.checked = false;
        } else {
            element.value = "";
        }
    });
}

// Füllt das Formular im Modal basierend auf den übergebenen Daten
function populateModalFields(modal, data) {
    const fields = modal.querySelectorAll(".modal-field");
    fields.forEach(field => {
        const fieldName = field.getAttribute("data-field");
        field.value = data ? data[fieldName] || "" : "";
    });
}

// Schließt das Modal und entfernt das Overlay
function closeModal() {
    document.getElementById("modal-container").style.display = "none";
    const overlay = document.querySelector(".modal-overlay");
    if (overlay) overlay.remove();
}

async function submitForm() {
    const form = document.getElementById("editForm");
    const data = {
        schema: currentSchema,
        table: currentTable,
        primaryKey: "", 
        columns: []
    };

    // Durchlaufe alle Eingabefelder und sammle Werte
    form.querySelectorAll("input, select, textarea").forEach((input) => {
        // Überspringe Felder ohne Namen
        if (!input.name) return;

        let value = input.value;

        // Setze den Primärschlüssel, falls das Feld als readonly markiert ist
        if (input.hasAttribute("readonly") && !data.primaryKey) {
            data.primaryKey = input.name;
        }

        // Spezifische Typ-Konvertierungen und Validierungen
        if (input.classList.contains("datepicker")) {
            // Datum in deutsches Format konvertieren
            const dateParts = value.split(".");
            if (dateParts.length === 3) {
                value = `${dateParts[2]}-${dateParts[1]}-${dateParts[0]}`;
            }
        } else if (input.classList.contains("timepicker") && !input.name.endsWith("_time")) {
            // Formatierung der Zeit für `time`-Eingaben im `HH:mm:00`-Format
            const [hours, minutes] = value.split(":");
            value = `${hours}:${minutes}:00`;
            data.columns.push({ name: input.name, value });
            return; // time-Eintrag wurde verarbeitet
        } else if (input.type === "checkbox") {
            // Boolean Werte
            value = input.checked;
        } else if (input.getAttribute("pattern") === "^[0-9]+(,[0-9]{1,2})?$") {
            // Float-Wert anpassen (Komma zu Punkt)
            value = value.replace(",", ".");
        }

        // Handling für `timestamp` mit getrennten Datum und Zeitfeldern
        if (input.name.endsWith("_date")) {
            const baseName = input.name.replace("_date", "");
            const timeInput = document.querySelector(`[name='${baseName}_time']`);
            if (timeInput && timeInput.value) {
                // Datum und Zeit kombinieren
                const dateValue = value.split("-");
                const timeValue = timeInput.value.split(":");
                value = new Date(
                    Date.UTC(
                        parseInt(dateValue[0]), // Jahr
                        parseInt(dateValue[1]) - 1, // Monat (0-basiert)
                        parseInt(dateValue[2]), // Tag
                        parseInt(timeValue[0]), // Stunde
                        parseInt(timeValue[1]) // Minute
                    )
                ).toISOString();
                data.columns.push({ name: baseName, value });

                // Explizit den zugehörigen `_time`-Eintrag überspringen
                timeInput.setAttribute("processed", "true"); // Markiere als verarbeitet
                return;
            }
        } else if (!input.name.endsWith("_time") && !input.hasAttribute("processed")) {
            // Standard-Fall: Feld hinzufügen, falls kein spezielles Zeitfeld und nicht bereits verarbeitet
            data.columns.push({ name: input.name, value });
        }
    });

    try {
        const response = await fetch("/api/save-record", {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify(data),
        });

        if (response.ok) {
            const result = await response.json();
            console.log("Speichern erfolgreich:", result);
            closeModal(); // Schließt das Modal nach erfolgreichem Speichern
            loadTableContent(); // Aktualisiert die Tabelle
        } else {
            const error = await response.text();
            console.error("Fehler beim Speichern:", error);
            alert(`Fehler beim Speichern: ${error}`);
        }
    } catch (error) { }
}