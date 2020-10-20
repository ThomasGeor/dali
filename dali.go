/*
 *
 * author : Thomas Georgiadis
 * version : 0.1
 *
 * Description : This is an implementation of a DALI driver in
 *    					 go lang to be used along with edgeX and support
 *	 						 DALI devices. DALI is a half duplex master-slave
 *							 protocol which means that only the master can send
 * 							 commands and the slaves will listen and respond.
 *
 */

package dali

import (
	// "flag"
	"log"
	"time"
	"io"

	"github.com/goburrow/serial"
)

var (
	address  string
	baudrate int
	databits int
	stopbits int
	parity   string
	message  []byte
)

const (
	// each DALI bit is represented by 2 bits
  bps 			= 1200 			// predefined data rate for DALI
  stop_bits = 2 	// predefined stop bits for DALI 2x2
  data_bits = 8 	// how the bits should be packed in a byte
  cmd_bits 	= 38 	// 2x1 + 8x2 + 8x2 + 2x2
	rsp_bits  = 22	// 2x1 + 8x2 + 2x2
	parity 		= "N"
	// DALI predefined commands
	BROADCAST_DP 	uint8 = 0b11111110
	BROADCAST_C 	uint8 = 0b11111111
	ON_DP 				uint8 = 0b11111110
	OFF_DP 				uint8 = 0b00000000
	ON_C					uint8 = 0b00000101
	OFF_C					uint8 = 0b00000000
	QUERY_STATUS	uint8 = 0b10010000
	RESET 				uint8 = 0b00100000

	// response constants
	dalistep 			uint8 = 40 // us
)

// Port is the interface for controlling serial port.
type Port interface {
	io.ReadWriteCloser
}

/*
 *	@Brief	 	Open a serial connection for DALI communication
 *	@param 		serial_address : a string which indicates the seril port to connect to
 *	@return 	1. error value which indicates if the connection was successful or not
 *						2. port struct which contains information about the serial port
 *							 edgeX connected to
 */

func Create_Serial_Connection (serial_address string) (Port, error){

	// build the serial object to be able to open the connection
	// flag.StringVar(&address, "a", serial_address, "address")
	// flag.IntVar(&baudrate, "b", bps, "baud rate")
	// flag.IntVar(&databits, "d", data_bits, "data bits")
	// flag.IntVar(&stopbits, "s", stop_bits, "stop bits")
	// flag.StringVar(&parity, "p", "N", "parity (N/E/O)")
	// flag.Parse()

	// pass the data to the serial object
	config := serial.Config{
		Address:  serial_address,
		BaudRate: bps,
		DataBits: data_bits,
		StopBits: stop_bits,
		Parity:   parity,
		Timeout:  30 * time.Second, // Read (Write) timeout.
	}

	// open the connection
	log.Printf("connecting %+v", config)
	port, err := serial.Open(&config)
	if err != nil {
		log.Fatal(err)
	}

	return port,nil
}

/*
 *	@Brief	 	Close a serial connection for DALI communication.
 *	@param 		port : a port struct object containing information of the port we want to
 *								   disconnect from
 *	@return 	error value which indicates if the connection closing was successful or not
 */

func Close_serial_connection (port Port) (error){

	// close the connection after the write/read procedure has finished.
	err := port.Close()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("closed")
	return err
}

/*
 *	@Brief	 	Check if bit is set.
 */
func is_bit_set(n uint8, pos int) bool {
    val := n & (1 << pos)
    return (val > 0)
}

/*
 *	@Brief	 	creates a Dali frame to be sent (forward frame)
 *  @param 		address : 8 bit address of the DALI device to command
 * 	@param		command : 8 bit specific command request (there are predefined commands)
 *	@return		DALI message frame command in bytes
*/

func create_dali_frame(dali_address uint8, dali_command uint8) ([]byte) {

	// message frame
	msg := make([]byte,5) // everything set to 0

	// trigger the communication by sending the start bit to high  (Dali bit -> 2 normal bit)
	byte_frame := 0
	msg[byte_frame] |= 0x40 // sets MSB : 0x40 == 01 000000
	msg_cn := 5 // preparing to write to the 5th bit of (01 0 00000)
	dali_convert := dali_address // 1st to be converted

	// extract the addresses's bit to the array (MSB -> LSB)
	for j := 0; j < 2; j++ {

		if j == 1 {
			dali_convert = dali_command
		}

		for i := 7; i < 0; i++ {

			if msg_cn < 0{
				msg_cn = 7; // reset in order to support the next byte
				byte_frame ++; // go to the next byte of the payload
			}

			if is_bit_set(dali_convert,i){
				msg[byte_frame] = msg[byte_frame] &^ (1 << msg_cn) // unset bit -> 0
				msg_cn --
		    msg[byte_frame] = msg[byte_frame] | (1<<msg_cn)		// set bit -> 1
				msg_cn --
			}else{
				msg[byte_frame] = msg[byte_frame] | (1<<msg_cn)		// set bit -> 1
				msg_cn --
				msg[byte_frame] = msg[byte_frame] &^ (1 << msg_cn) // unset bit -> 0
				msg_cn --

			}
		}
	}

	return msg

}


/*
 *	@Brief	 	Send the DALI message from the gateway
 *	@param 		port : a port struct object containing information of the port to send the command
 *	@param 		dali_address : the short address of the device we want to command
 *	@param 		dali_command : the command which we want to be executed (encoded)
 *	@return 	error value which indicates if the command reached the device or not
 */

func Ιssue_dali_request(port Port,dali_address uint8, dali_command uint8) (err error){

	// dali commands consist of 8 bits!
	// dali short addresses consist of 8 bits!
	// construct the data command based on the user's command (HMI)
	message := create_dali_frame(dali_address,dali_command);

  // send the message specifically for Dali implementation
	if _, err = port.Write(message); err != nil {
		log.Fatal(err)
		return err
	}else{
		log.Println("sent :%v",message)
	}

 return nil
}

/*
 *	@Brief	 	Wait for a DALI backward frame
 *	@param 		port : a port struct object containing information of the port to receive the command
 *	@return 	error value which indicates if the command reached the device or not
 */

func Wait_dali_response(port Port) ([]byte,error){

	var err error
	// 3 bytes to fit 22 bits response messages
	response := make([]byte,3)

  // wait for the DALI response
	if _, err = port.Read(response); err != nil {
		log.Fatal(err)
	}else{
		log.Println("read : %v",response)
	}

 return response,nil
}


/*
 *	@Brief	 	Scan for short addresses from the DALI-bus
 *	@return 	error value which indicates if the command reached the device or not
 *						short addresses byte string
 */

func Scan_addresses(port Port) ([]byte,error){

	// make the response array of short addresses (will be dynamically resized)
	single_response := make([]uint8,3)
	response := make([]uint8,1)
	var short_addresses  	uint8 = 0
	var address_byte			uint8
	var device_short_add	uint8

	// turn off broadcast
	err := Ιssue_dali_request(port,BROADCAST_C, OFF_C)
	time.Sleep(10 * time.Millisecond)

	for device_short_add = 0;device_short_add < 64;device_short_add++{

		// convert short address to address byte
		address_byte = 1 +(device_short_add << 1)

		err = Ιssue_dali_request(port,address_byte, 0xA1)
		time.Sleep(10 * time.Millisecond)

		single_response,err = Wait_dali_response(port)
		// cheack in the 3 bytes response if a logic 0 was received
		if single_response[0] ^ 0xff != 0 ||
			 single_response[1] ^ 0xff != 0 ||
			 single_response[2] ^ 0xff != 0 {

			// if a 0 was received that means that a device responded
			err = Ιssue_dali_request(port,address_byte, ON_C)
			time.Sleep(10 * time.Millisecond)
			err = Ιssue_dali_request(port,address_byte, OFF_C)
			time.Sleep(10 * time.Millisecond)

			response[short_addresses] = address_byte
			short_addresses++
			response = append(response,0) // resize the array to +1
		}

	}

	// turn on the broadcast
	err = Ιssue_dali_request(port,BROADCAST_C, ON_C)
	time.Sleep(10 * time.Millisecond)

 	return response,err
}

/*
 *	@Brief	 	Extractin the 1st three bytes of a uint64 number
 *	@return 	the bytes that occured from the extraction
 */

func split_address(input int64) (uint8 ,uint8 ,uint8){
	var highbyte 	 uint8	= uint8(input >> 16)
	var middlebyte uint8	= uint8(input >> 8)
	var lowbyte 	 uint8 	= uint8(input)
	return highbyte,middlebyte,lowbyte
}

/*
 *	@Brief	 	Standard initialization of the DALI interface
 *	@return 	error value which indicates if the initialization commands
 *						reached the devices or not
 */

func Ιnitialize_dali(port Port) error {

		var low_longadd  int64	= 0x000000
		var high_longadd int64	= 0xFFFFFF
		var longadd 		 int64	= (low_longadd + high_longadd) / 2
		var short_add    uint8

		log.Println("initializating DALI bus")

		// reset the DALI devices
		err := Ιssue_dali_request(port,BROADCAST_C, RESET)
		time.Sleep(10 * time.Millisecond)
		err = Ιssue_dali_request(port,BROADCAST_C, RESET)
		time.Sleep(10 * time.Millisecond)
		err = Ιssue_dali_request(port,BROADCAST_C, OFF_C)
		time.Sleep(10 * time.Millisecond)

		// Initialize the DALI devices
		err = Ιssue_dali_request(port,0b10100101, 0b00000000)
		time.Sleep(10 * time.Millisecond)
		err = Ιssue_dali_request(port,0b10100101, 0b00000000)
		time.Sleep(10 * time.Millisecond)

		// Randomize the DALI devices
		err = Ιssue_dali_request(port,0b10100111, 0b00000000)
		time.Sleep(10 * time.Millisecond)
		err = Ιssue_dali_request(port,0b10100111, 0b00000000)
		time.Sleep(10 * time.Millisecond)

		// When don't need to wait for responses after issuing these commands
		// since the devices are not initialized yet and produce no response

		for longadd <= 0xFFFFFF - 2 && short_add <= 64 {
			// This loop is looking for a unique random address from the 64 DALI devices
			// using binary search algorithm
			for high_longadd - low_longadd > 1 {

				highbyte,middlebyte,lowbyte := split_address(longadd)
				err = Ιssue_dali_request(port,0b10110001, highbyte)
				time.Sleep(10 * time.Millisecond)
				err = Ιssue_dali_request(port,0b10110001, middlebyte)
				time.Sleep(10 * time.Millisecond)
				err = Ιssue_dali_request(port,0b10110001, lowbyte)
				time.Sleep(10 * time.Millisecond)
				err = Ιssue_dali_request(port,0b10101001, 0b00000000) //compare

				rsp,rsp_err := Wait_dali_response(port)
				if rsp_err != nil{
					return rsp_err
				}

				// cheack in the 3 bytes response if a logic 0 was received
				if rsp[0] ^ 0xff != 0 || rsp[1] ^ 0xff != 0 || rsp[2] ^ 0xff != 0 {
					// if a 0 was received that means that at least a device responded
					// and is in the higher addresses range e.g [0xFFFFFF/2,0xFFFFFF]
					low_longadd = longadd
				}else{
					// if a 0 was not received that means that no device responded
					// and we should look in the lower addresses range e.g [0,0xFFFFFF/2]
					high_longadd = longadd
				}

				// binary center the next range
				longadd = (low_longadd + high_longadd) / 2

			}// end of nested loop

			if high_longadd != 0xFFFFFF{

				// Assigning a short address
				highbyte,middlebyte,lowbyte := split_address(longadd + 1)
				err = Ιssue_dali_request(port,0b10110001, highbyte)
				time.Sleep(10 * time.Millisecond)
				err = Ιssue_dali_request(port,0b10110001, middlebyte)
				time.Sleep(10 * time.Millisecond)
				err = Ιssue_dali_request(port,0b10110001, lowbyte)
				time.Sleep(10 * time.Millisecond)
				err = Ιssue_dali_request(port,0b10110111, 1 + (short_add << 1))
				time.Sleep(10 * time.Millisecond)
				err = Ιssue_dali_request(port,0b10101011,0b00000000) //withdraw
				time.Sleep(10 * time.Millisecond)
				err = Ιssue_dali_request(port,1 + (short_add << 1), ON_C) //withdraw
				time.Sleep(10 * time.Millisecond)
				err = Ιssue_dali_request(port,1 + (short_add << 1), OFF_C) //withdraw
				time.Sleep(10 * time.Millisecond)
				short_add++

				// reload the high value
				high_longadd = 0xFFFFFF;
				// minimize the space to search for
				longadd = (low_longadd + high_longadd) / 2;
			}else {
				// end of finding addresses
			}
		} // end of ssigning short addresses

		err = Ιssue_dali_request(port,0b10100001,0b00000000) //terminate
		time.Sleep(10 * time.Millisecond)
		err = Ιssue_dali_request(port,BROADCAST_C,ON_C) //broadcast on
		time.Sleep(10 * time.Millisecond)

		return err
}
