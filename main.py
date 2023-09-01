import os
import keyring
import requests
from PIL import Image, ImageDraw, ImageFont


# Load credentials from keychain
printer_api_key = 'Bearer ' + keyring.get_password("helpdesk_printer", "")


def generate(spare_num):
    # Generate image of computer information with QR code
    template_img = Image.open(os.path.join(
        os.path.dirname(__file__), 'assets/template.png'))
    draw = ImageDraw.Draw(template_img)
    font_heading = ImageFont.truetype('assets/Arial Bold.ttf', size=245)
    # Add computer information to image
    draw.text((78, 192), f'Spare {spare_num}',
              font=font_heading, fill='black')
    # Save image to temporary file
    info_image_path = f'/tmp/tmp.upload-{spare_num}.png'
    template_img.save(info_image_path)


def print(printer_api_key, spare_num):
    # Print image to network printer
    headers = {'Authorization': f'{printer_api_key}'}
    files = {'file': open(f'/tmp/tmp.upload-{spare_num}.png', 'rb')}
    try:
        response = requests.post(
            'http://172.19.10.12:5001/print_image', headers=headers, files=files)
        if response.status_code != 200:
            raise ValueError(f"Failed to print?")
    except:
        raise ValueError(f"Failed to connect to printer")
    # Remove temporary file after printing
    os.remove(f'/tmp/tmp.upload-{spare_num}.png')


# set numbers to print here, eg `(3, 5)` will print numbers 3-4 (range)
for i in range(7, 8):
    spare_num = f"{i:02d}"
    generate(spare_num)
    print(printer_api_key, spare_num)
