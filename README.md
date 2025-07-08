# ATC Simulator Game

This is a 2D Air Traffic Control (ATC) simulation game developed in Go. It features a client-server architecture to simulate managing aircraft within a defined airspace, handling flight plans, issuing commands, and preventing conflicts.

## Features

The game includes:

*   **2D Simulation:** Visuals for aircraft and airspace elements.
*   **Client-Server Architecture:** Will support networked gameplay or simulation (coming soon).
*   **Aircraft Simulation:** Models aircraft behavior and movement.
*   **Airspace & Airport Management:** Defines the game environment, including airports and sectors.
*   **Flight Plan Management:** Tracks aircraft routes and intentions.
*   **Conflict Detection:** Identifies potential collisions between aircraft (simple).
*   **ATC Commands:** Allows the player to issue instructions to aircraft (likely via text input).
*   **Radio Communication Simulation:** Basic handling of radio messages.
*   **Scenarios:** Support for different game setups or levels.
*   **Assets:** Includes basic audio, fonts, and aircraft images.

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

### Prerequisites

*   Go (version 1.23 or higher)

### Building

The project includes a `Makefile` for easy building.

1.  Clone the repository (assuming it's in a VCS).
2.  Navigate to the project root directory.
3.  Run the make command to build the client and potentially the server:

    ```bash
    make build
    ```

    This should produce the client executable `bin/atc-sim-client`. Building the server might require a separate command or is included in the `make build`.

## How to Run

1.  Start the server component (details on starting the server are not explicitly in the file list, but it likely resides within the `cmd/server` directory).
2.  Run the client executable:

    ```bash
    bin/atc-sim-client
    ```

## How to Play

The game involves managing air traffic within a simulated airspace. Use the in-game interface (involving text commands) to guide aircraft, manage their altitudes and headings, and ensure they follow their flight plans without colliding.

## Licence

This project is licensed under the terms specified in the [LICENCE](LICENCE) file.
