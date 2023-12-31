import os
import json
import requests
from PIL import Image, ImageDraw, ImageFont


# Load credentials from json file
with open('/Users/Shared/secrets.json') as f:
    data = json.load(f)
    api_token = 'Bearer ' + data['api_token']


def generate(spare_num):
    # Generate image of computer information with QR code
    template_img = Image.open(os.path.join(
        os.path.dirname(__file__), 'assets/template.png'))
    draw = ImageDraw.Draw(template_img)
    font_heading = ImageFont.truetype('assets/Arial Bold.ttf', size=245)
    # Add computer information to image
    draw.text((44, 84), f'Spare {spare_num}',
              font=font_heading, fill='black')
    # Save image to temporary file
    info_image_path = f'/tmp/tmp.upload-{spare_num}.png'
    template_img.save(info_image_path)


def print(api_token, spare_num):
    # Print image to network printer
    headers = {'Authorization': f'{api_token}'}
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


for i in range(13, 16):
    spare_num = f"{i:02d}"
    generate(spare_num)
    print(api_token, spare_num)
