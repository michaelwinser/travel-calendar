# **Technical Foundation: The Kinetic Ledger**

## **1\. Data Model Strategy**

The data model needs to support both "point-in-time" events and "spanning" activities while allowing for efficient conflict resolution.

### **1.1 Entities**

#### **Location**

Represents a physical geographic coordinate or named area.

* `id`: UUID  
* `name`: String (e.g., "London, UK", "Home")  
* `type`: Enum (HOME, WORK, TRANSIT, AWAY)  
* `timezone`: String (IANA format)

#### **Activity**

The primary unit of planning.

* `id`: UUID  
* `title`: String  
* `description`: Text (optional)  
* `type`: Enum (TRAVEL, STAY, CONFERENCE, VACATION, COMMITMENT, BASELINE)  
* `start_date`: Date  
* `end_date`: Date  
* `location_id`: ForeignKey(Location)  
* `is_all_day`: Boolean  
* `metadata`: JSON (e.g., flight numbers, hotel confirmation)  
* `source`: Enum (MANUAL, GOOGLE\_CALENDAR, SYSTEM)

#### **Conflict**

Computed state identifying incompatible activities.

* `id`: UUID  
* `activity_ids`: List\[UUID\] (references to the conflicting activities)  
* `date`: Date  
* `severity`: Enum (CRITICAL, WARNING, INFO)  
* `resolution_status`: Enum (OPEN, IGNORED, RESOLVED)

---

## **2\. Enumerated Use Cases (Acceptance Criteria)**

### **UC-1001: Multi-Day Activity Creation**

* **Description:** User creates an activity that spans multiple days (e.g., a trip to Seattle).  
* **Acceptance Criteria:**  
  * Activity persists in the database with correct start/end dates.  
  * Grid view renders a continuous highlighter bar across the specified days.  
  * Location is correctly associated with the date range.

### **UC-1002: Automatic Conflict Detection**

* **Description:** System identifies when a local commitment (Type: COMMITMENT) occurs while the user is in a different Location Type (Type: AWAY).  
* **Acceptance Criteria:**  
  * System flags the specific date with a conflict warning.  
  * Grid view shows the red exclamation triangle in the cell corner.  
  * List/Agenda views highlight the row with a red background wash.

### **UC-1003: Location Overlay Toggle**

* **Description:** User toggles the display of location labels within the calendar grid.  
* **Acceptance Criteria:**  
  * Grid cells dynamically show/hide location text based on toggle state.  
  * Layout remains stable and scannable regardless of toggle state.

### **UC-1004: Google Calendar Import & Noise Filtering**

* **Description:** System imports events from GCal but ignores virtual meetings.  
* **Acceptance Criteria:**  
  * Events with "Zoom", "Meet", or "Teams" links in the location field are ignored or flagged as non-physical.  
  * "Working From" locations are mapped to the daily Location state.

### **UC-1005: Infinite Scroll Navigation**

* **Description:** User scrolls vertically through weeks without artificial month boundaries.  
* **Acceptance Criteria:**  
  * Weeks are displayed in a continuous unbroken sequence.  
  * Month/Year labels in the sidebar/header update accurately as the view scrolls.

---

## **3\. CLI Specification (`kl-cli`)**

To support power users and automation, a CLI provides direct access to the ledger.

### **Commands**

#### **`kl add [title] --from [date] --to [date] --loc [location] --type [type]`**

* **Example:** `kl add "European Summit" --from 2024-10-04 --to 2024-10-07 --loc "Brussels" --type conference`  
* **Output:** Success message with the generated Activity ID and any immediate conflict warnings.

#### **`kl list --month [month] --year [year]`**

* **Example:** `kl list --month oct --year 2024`  
* **Output:** A text-based linear representation of the month, highlighting activities and location zones.

#### **`kl conflicts`**

* **Example:** `kl conflicts`  
* **Output:** A list of all unresolved conflicts, including the specific activities and dates involved.

#### **`kl check [date]`**

* **Example:** `kl check 2024-10-14`  
* **Output:** "Location: London (Away). Activities: Tech Summit, Flight LHR \-\> JFK. Status: Stable."

#### **`kl export --format [json|csv|ical]`**

* **Output:** Exports the ledger data for use in other tools or sharing.

