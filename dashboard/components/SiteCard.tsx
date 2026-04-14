import { Delete, Ellipsis, Trash, Trash2 } from 'lucide-react';
import React from 'react';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from './ui/dropdown-menu';
// import { ExternalLink, BarChart3 } from 'lucide-react'; // Optional icons

const SiteCard = ({name, url, visitorCount }:{ name: string, url: string, visitorCount: string }) => {
  return (
    <div className="group relative w-72 h-40 bg-[#F8FAFB] border border-gray-200 rounded-md p-2 
                    transition-all duration-300 ease-in-out hover:shadow-lg hover:border-black cursor-pointer">
      
      <div className="flex flex-col h-full justify-between bg-card p-3">
        {/* Header Section */}
        <div>
          <div className="flex justify-between items-center">
            <h3 className="text-2xl font-serif font-medium text-gray-900 group-hover:text-black">
              {name}
            </h3>
            {/*<ExternalLink size={16} className="text-gray-400 opacity-0 group-hover:opacity-100 transition-opacity" />*/}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button className='bg-zinc-50 flex justify-center items-center rounded-xs cursor-pointer hover:bg-zinc-200 border border-border'>
                  <Ellipsis className='w-4 h-4'/>
                </button>
              </DropdownMenuTrigger>
              
              <DropdownMenuContent className='rounded-sm'>
                <DropdownMenuItem className='rounded-xs cursor-pointer'>
                  <Trash2/>
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
          <p className="text-base text-gray-500 font-mono mt-1 h-12 overflow-hidden hover:underline">{url}</p>
        </div>

        {/* Analytics Preview / Footer */}
        <div className="flex items-center gap-2 text-gray-400 group-hover:text-black transition-colors">
          {/*<BarChart3 size={18} />*/}
          <span className="text-base font-semibold font-serif tracking-widest uppercase">
            {visitorCount} Visitors
          </span>
        </div>
      </div>

      {/* Subtle Bottom Accent */}
      <div className="absolute bottom-0 left-0 w-full h-1 bg-black transform scale-x-0 group-hover:scale-x-100 transition-transform duration-300 rounded-b-xl" />
    </div>
  );
};

export default SiteCard;