import os
import json
import requests
import keyring
from PIL import Image, ImageDraw, ImageFont


# Load credentials from keychain
printer_api_key = 'Bearer ' + keyring.get_password("helpdesk_printer", "")


def generate(spare_num):
    tag_text = f'SAC {spare_num}'
    # Load the template image
    template_img = Image.open(os.path.join(
        os.path.dirname(__file__), 'assets/template.png'))
    draw = ImageDraw.Draw(template_img)

    # Template dimensions
    template_width, template_height = template_img.size

    # Initial font size
    font_size = 245

    # Create font object
    font = ImageFont.truetype('assets/Arial Bold.ttf', size=font_size)

    # Get width of the text
    text_width = draw.textlength(tag_text, font=font)
    # Height of the text (approximated by font size)
    text_height = font_size

    # Decrease font size until the text fits within the boundaries
    while text_width > 1181 or text_height > 566:
        font_size -= 1
        font = ImageFont.truetype('assets/Arial Bold.ttf', size=font_size)
        text_width = draw.textlength(tag_text, font=font)
        text_height = font_size

    # Calculate starting coordinates to center the text
    x = (template_width - text_width) / 2
    y = (template_height - text_height) / 2

    # Draw text
    draw.text((x, y), tag_text, font=font, fill='black')

    # Save image to temporary file
    info_image_path = f'/tmp/tmp.upload-{spare_num}.png'
    template_img.save(info_image_path)


def print_image(printer_api_key, spare_num):
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
for i in range(13, 15):
    spare_num = f"{i:02d}"
    generate(spare_num)
    print_image(printer_api_key, spare_num)
