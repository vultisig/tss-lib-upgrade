use cait_sith::protocol::{Protocol, Action, ProtocolError, Participant, MessageData};
use k256::{elliptic_curve::{CurveArithmetic, scalar::FromUintUnchecked}, Secp256k1, U256};

use std::{collections::HashMap, process::Output};
use std::sync::Arc;
use tokio::sync::Mutex;
use tokio::io::{AsyncReadExt, AsyncWriteExt, BufReader};

use tokio::net::{TcpListener, TcpStream};
use tokio::sync::mpsc;
use std::net::SocketAddr;
use std::io::{self, Write};
use std::error::Error;

struct MyProtocol(Box<dyn Protocol<Output = String>>);

impl cait_sith::protocol::Protocol for MyProtocol {
    type Output = String;

    fn poke(&mut self) -> Result<Action<Self::Output>, ProtocolError> {
        Ok(Action::Wait)
    }

    fn message(&mut self, _from: Participant, _message: MessageData) {
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn Error + Send + Sync>> {
    println!("Enter your listening address (e.g., 127.0.0.1:8080):");
    let mut input = String::new();
    io::stdin().read_line(&mut input)?;
    let listen_addr: SocketAddr = input.trim().parse()?;

    let listener = TcpListener::bind(listen_addr).await?;
    println!("Listening on: {}", listen_addr);

    let (tx, mut rx) = mpsc::channel::<(String, SocketAddr)>(32);
    let tx_clone = tx.clone();

    // Spawn a task to handle incoming connections
    tokio::spawn(async move {
        while let Ok((mut socket, addr)) = listener.accept().await {
            let tx = tx_clone.clone();
            tokio::spawn(async move {
                let (mut reader, mut writer) = socket.split();
                let mut buf = [0u8; 1024];
                loop {
                    match reader.read(&mut buf).await {
                        Ok(0) => break,
                        Ok(n) => {
                            let message = String::from_utf8_lossy(&buf[..n]).to_string();
                            tx.send((message, addr)).await.unwrap();
                        }
                        Err(_) => break,
                    }
                }
            });
        }
    });

    // Handle user input and message sending
    tokio::spawn(async move {
        loop {
            println!("Enter the address to connect (or 'q' to quit):");
            let mut input = String::new();
            io::stdin().read_line(&mut input)?;
            let input = input.trim();

            if input == "q" {
                break;
            }

            let addr: SocketAddr = match input.parse() {
                Ok(addr) => addr,
                Err(_) => {
                    println!("Invalid address format. Please try again.");
                    continue;
                }
            };

            match TcpStream::connect(addr).await {
                Ok(mut stream) => {
                    println!("Connected to: {}", addr);
                    loop {
                        println!("Enter message (or 'q' to disconnect):");
                        let mut message = String::new();
                        io::stdin().read_line(&mut message)?;
                        let message = message.trim();

                        if message == "q" {
                            break;
                        }

                        if let Err(e) = stream.write_all(message.as_bytes()).await {
                            println!("Failed to send message: {}", e);
                            break;
                        }
                    }
                }
                Err(e) => println!("Failed to connect: {}", e),
            }
        }
        Ok::<(), Box<dyn Error + Send + Sync>>(())
    });

    // Handle incoming messages
    while let Some((message, addr)) = rx.recv().await {
        println!("Received from {}: {}", addr, message);
    }

    Ok(())
}

